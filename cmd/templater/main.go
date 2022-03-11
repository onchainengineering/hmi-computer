package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
	"google.golang.org/api/idtoken"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/sloghuman"
	"github.com/coder/coder/coderd"
	"github.com/coder/coder/coderd/tunnel"
	"github.com/coder/coder/codersdk"
	"github.com/coder/coder/database"
	"github.com/coder/coder/database/databasefake"
	"github.com/coder/coder/provisioner/terraform"
	"github.com/coder/coder/provisionerd"
	"github.com/coder/coder/provisionersdk"
	"github.com/coder/coder/provisionersdk/proto"

	"github.com/gohugoio/hugo/parser/pageparser"
)

func main() {
	var rawParameters []string
	cmd := &cobra.Command{
		Use: "templater",
		RunE: func(cmd *cobra.Command, args []string) error {
			parameters := make([]codersdk.CreateParameterRequest, 0)
			for _, parameter := range rawParameters {
				parts := strings.SplitN(parameter, "=", 2)
				parameters = append(parameters, codersdk.CreateParameterRequest{
					Name:              parts[0],
					SourceValue:       parts[1],
					SourceScheme:      database.ParameterSourceSchemeData,
					DestinationScheme: database.ParameterDestinationSchemeProvisionerVariable,
				})
			}
			return parse(cmd, args, parameters)
		},
	}
	cmd.Flags().StringArrayVarP(&rawParameters, "parameter", "p", []string{}, "Specify parameters to pass in a template.")
	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}

func parse(cmd *cobra.Command, args []string, parameters []codersdk.CreateParameterRequest) error {
	srv := httptest.NewUnstartedServer(nil)
	srv.Config.BaseContext = func(_ net.Listener) context.Context {
		return cmd.Context()
	}
	srv.Start()
	serverURL, err := url.Parse(srv.URL)
	if err != nil {
		return err
	}
	accessURL, errCh, err := tunnel.New(cmd.Context(), srv.URL)
	go func() {
		err := <-errCh
		if err != nil {
			panic(err)
		}
	}()
	accessURLParsed, err := url.Parse(accessURL)
	if err != nil {
		return err
	}
	var closeWait func()
	validator, err := idtoken.NewValidator(cmd.Context())
	if err != nil {
		return err
	}
	logger := slog.Make(sloghuman.Sink(cmd.OutOrStdout()))
	srv.Config.Handler, closeWait = coderd.New(&coderd.Options{
		AccessURL:            accessURLParsed,
		Logger:               logger,
		Database:             databasefake.New(),
		Pubsub:               database.NewPubsubInMemory(),
		GoogleTokenValidator: validator,
	})

	client := codersdk.New(serverURL)
	daemonClose, err := newProvisionerDaemon(cmd.Context(), client, logger)
	if err != nil {
		return err
	}
	defer daemonClose.Close()

	created, err := client.CreateFirstUser(cmd.Context(), codersdk.CreateFirstUserRequest{
		Email:        "templater@coder.com",
		Username:     "templater",
		Organization: "templater",
		Password:     "insecure",
	})
	if err != nil {
		return err
	}
	auth, err := client.LoginWithPassword(cmd.Context(), codersdk.LoginWithPasswordRequest{
		Email:    "templater@coder.com",
		Password: "insecure",
	})
	if err != nil {
		return err
	}
	client.SessionToken = auth.SessionToken

	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	content, err := provisionersdk.Tar(dir)
	if err != nil {
		return err
	}
	resp, err := client.Upload(cmd.Context(), codersdk.ContentTypeTar, content)
	if err != nil {
		return err
	}

	before := time.Now()
	version, err := client.CreateProjectVersion(cmd.Context(), created.OrganizationID, codersdk.CreateProjectVersionRequest{
		ProjectID:       nil,
		StorageMethod:   database.ProvisionerStorageMethodFile,
		StorageSource:   resp.Hash,
		Provisioner:     database.ProvisionerTypeTerraform,
		ParameterValues: parameters,
	})
	if err != nil {
		return err
	}
	logs, err := client.ProjectVersionLogsAfter(cmd.Context(), version.ID, before)
	if err != nil {
		return err
	}
	for {
		log, ok := <-logs
		if !ok {
			break
		}
		fmt.Printf("terraform (%s): %s\n", log.Level, log.Output)
	}
	version, err = client.ProjectVersion(cmd.Context(), version.ID)
	if err != nil {
		return err
	}
	if version.Job.Status != codersdk.ProvisionerJobSucceeded {
		return xerrors.Errorf("Job wasn't successful, it was %q. Check the logs!", version.Job.Status)
	}

	schemas, err := client.ProjectVersionSchema(cmd.Context(), version.ID)
	if err != nil {
		return err
	}
	resources, err := client.ProjectVersionResources(cmd.Context(), version.ID)
	if err != nil {
		return err
	}

	readme, err := os.OpenFile(filepath.Base(dir)+".md", os.O_RDONLY, 0600)
	if err != nil {
		return err
	}
	defer readme.Close()
	frontMatter, err := pageparser.ParseFrontMatterAndContent(readme)
	if err != nil {
		return err
	}
	name, exists := frontMatter.FrontMatter["name"]
	if !exists {
		return xerrors.New("front matter must contain name")
	}
	description, exists := frontMatter.FrontMatter["description"]
	if !exists {
		return xerrors.Errorf("front matter must contain description")
	}

	for index, resource := range resources {
		resource.ID = uuid.UUID{}
		resource.JobID = uuid.UUID{}
		resource.CreatedAt = time.Time{}

		if resource.Agent != nil {
			resource.Agent.ID = uuid.UUID{}
			resource.Agent.ResourceID = uuid.UUID{}
			resource.Agent.CreatedAt = time.Time{}
		}
		resources[index] = resource
	}

	for index, schema := range schemas {
		schema.ID = uuid.UUID{}
		schema.JobID = uuid.UUID{}
		schema.CreatedAt = time.Time{}
		schemas[index] = schema
	}
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Name < resources[j].Name
	})
	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].Name < schemas[j].Name
	})

	template := codersdk.Template{
		ID:                            filepath.Base(dir),
		Name:                          name.(string),
		Description:                   description.(string),
		ProjectVersionParameterSchema: schemas,
		Resources:                     resources,
	}

	data, err := json.MarshalIndent(template, "", "\t")
	if err != nil {
		return err
	}
	os.WriteFile(filepath.Base(dir)+".json", data, 0600)

	project, err := client.CreateProject(cmd.Context(), created.OrganizationID, codersdk.CreateProjectRequest{
		Name:      "test",
		VersionID: version.ID,
	})
	if err != nil {
		return err
	}

	workspace, err := client.CreateWorkspace(cmd.Context(), created.UserID, codersdk.CreateWorkspaceRequest{
		ProjectID: project.ID,
		Name:      "example",
	})
	if err != nil {
		return err
	}

	build, err := client.CreateWorkspaceBuild(cmd.Context(), workspace.ID, codersdk.CreateWorkspaceBuildRequest{
		ProjectVersionID: version.ID,
		Transition:       database.WorkspaceTransitionStart,
	})
	if err != nil {
		return err
	}
	logs, err = client.WorkspaceBuildLogsAfter(cmd.Context(), build.ID, before)
	if err != nil {
		return err
	}
	for {
		log, ok := <-logs
		if !ok {
			break
		}
		fmt.Printf("terraform (%s): %s\n", log.Level, log.Output)
	}

	resources, err = client.WorkspaceResourcesByBuild(cmd.Context(), build.ID)
	if err != nil {
		return err
	}
	for _, resource := range resources {
		if resource.Agent == nil {
			continue
		}
		err = awaitAgent(cmd.Context(), client, resource)
		if err != nil {
			return err
		}
	}

	build, err = client.CreateWorkspaceBuild(cmd.Context(), workspace.ID, codersdk.CreateWorkspaceBuildRequest{
		ProjectVersionID: version.ID,
		Transition:       database.WorkspaceTransitionDelete,
	})
	if err != nil {
		return err
	}
	logs, err = client.WorkspaceBuildLogsAfter(cmd.Context(), build.ID, before)
	if err != nil {
		return err
	}
	for {
		log, ok := <-logs
		if !ok {
			break
		}
		fmt.Printf("terraform (%s): %s\n", log.Level, log.Output)
	}

	daemonClose.Close()
	srv.Close()
	closeWait()
	return nil
}

func awaitAgent(ctx context.Context, client *codersdk.Client, resource codersdk.WorkspaceResource) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			resource, err := client.WorkspaceResource(ctx, resource.ID)
			if err != nil {
				return err
			}
			if resource.Agent.UpdatedAt.IsZero() {
				continue
			}
			return nil
		}
	}
}

func newProvisionerDaemon(ctx context.Context, client *codersdk.Client, logger slog.Logger) (io.Closer, error) {
	terraformClient, terraformServer := provisionersdk.TransportPipe()
	go func() {
		err := terraform.Serve(ctx, &terraform.ServeOptions{
			ServeOptions: &provisionersdk.ServeOptions{
				Listener: terraformServer,
			},
			Logger: logger,
		})
		if err != nil {
			panic(err)
		}
	}()
	tempDir, err := ioutil.TempDir("", "provisionerd")
	if err != nil {
		return nil, err
	}
	return provisionerd.New(client.ListenProvisionerDaemon, &provisionerd.Options{
		Logger:         logger,
		PollInterval:   50 * time.Millisecond,
		UpdateInterval: 500 * time.Millisecond,
		Provisioners: provisionerd.Provisioners{
			string(database.ProvisionerTypeTerraform): proto.NewDRPCProvisionerClient(provisionersdk.Conn(terraformClient)),
		},
		WorkDirectory: tempDir,
	}), nil
}
