package coderd

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/moby/moby/pkg/namesgenerator"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"

	"github.com/coder/coder/coderd/autobuild/schedule"
	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/coderd/httpapi"
	"github.com/coder/coder/coderd/httpmw"
	"github.com/coder/coder/coderd/rbac"
	"github.com/coder/coder/codersdk"
)

func (api *api) workspace(rw http.ResponseWriter, r *http.Request) {
	workspace := httpmw.WorkspaceParam(r)

	if !api.Authorize(rw, r, rbac.ActionRead,
		rbac.ResourceWorkspace.InOrg(workspace.OrganizationID).WithOwner(workspace.OwnerID.String()).WithID(workspace.ID.String())) {
		return
	}

	// The `deleted` query parameter (which defaults to `false`) MUST match the
	// `deleted` field on the workspace otherwise you will get a 410 Gone.
	var (
		deletedStr = r.URL.Query().Get("deleted")
		deleted    = false
	)
	if deletedStr != "" {
		var err error
		deleted, err = strconv.ParseBool(deletedStr)
		if err != nil {
			httpapi.Write(rw, http.StatusBadRequest, httpapi.Response{
				Message: fmt.Sprintf("invalid bool for 'deleted' query param: %s", err),
			})
			return
		}
	}
	if workspace.Deleted != deleted {
		if workspace.Deleted {
			httpapi.Write(rw, http.StatusGone, httpapi.Response{
				Message: fmt.Sprintf("workspace %q was deleted, you can view this workspace by specifying '?deleted=true' and trying again", workspace.ID.String()),
			})
			return
		}

		httpapi.Write(rw, http.StatusBadRequest, httpapi.Response{
			Message: fmt.Sprintf("workspace %q is not deleted, please remove '?deleted=true' and try again", workspace.ID.String()),
		})
		return
	}

	build, err := api.Database.GetWorkspaceBuildByWorkspaceIDWithoutAfter(r.Context(), workspace.ID)
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("get workspace build: %s", err),
		})
		return
	}
	var (
		group    errgroup.Group
		job      database.ProvisionerJob
		template database.Template
		owner    database.User
	)
	group.Go(func() (err error) {
		job, err = api.Database.GetProvisionerJobByID(r.Context(), build.JobID)
		return err
	})
	group.Go(func() (err error) {
		template, err = api.Database.GetTemplateByID(r.Context(), workspace.TemplateID)
		return err
	})
	group.Go(func() (err error) {
		owner, err = api.Database.GetUserByID(r.Context(), workspace.OwnerID)
		return err
	})
	err = group.Wait()
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("fetch resource: %s", err),
		})
		return
	}

	httpapi.Write(rw, http.StatusOK,
		convertWorkspace(workspace, convertWorkspaceBuild(build, convertProvisionerJob(job)), template, owner))
}

func (api *api) workspacesByOrganization(rw http.ResponseWriter, r *http.Request) {
	organization := httpmw.OrganizationParam(r)
	roles := httpmw.UserRoles(r)
	workspaces, err := api.Database.GetWorkspacesByOrganizationID(r.Context(), database.GetWorkspacesByOrganizationIDParams{
		OrganizationID: organization.ID,
		Deleted:        false,
	})
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("get workspaces: %s", err),
		})
		return
	}

	allowedWorkspaces := make([]database.Workspace, 0)
	for _, ws := range workspaces {
		ws := ws
		err = api.Authorizer.ByRoleName(r.Context(), roles.ID.String(), roles.Roles, rbac.ActionRead,
			rbac.ResourceWorkspace.InOrg(ws.OrganizationID).WithOwner(ws.OwnerID.String()).WithID(ws.ID.String()))
		if err == nil {
			allowedWorkspaces = append(allowedWorkspaces, ws)
		}
	}

	apiWorkspaces, err := convertWorkspaces(r.Context(), api.Database, allowedWorkspaces)
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("convert workspaces: %s", err),
		})
		return
	}
	httpapi.Write(rw, http.StatusOK, apiWorkspaces)
}

func (api *api) workspacesByUser(rw http.ResponseWriter, r *http.Request) {
	user := httpmw.UserParam(r)
	roles := httpmw.UserRoles(r)

	allWorkspaces := make([]database.Workspace, 0)
	userWorkspaces, err := api.Database.GetWorkspacesByOwnerID(r.Context(), database.GetWorkspacesByOwnerIDParams{
		OwnerID: user.ID,
	})
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("get workspaces for user: %s", err),
		})
		return
	}
	for _, ws := range userWorkspaces {
		ws := ws
		err = api.Authorizer.ByRoleName(r.Context(), user.ID.String(), roles.Roles, rbac.ActionRead,
			rbac.ResourceWorkspace.InOrg(ws.OrganizationID).WithOwner(ws.OwnerID.String()).WithID(ws.ID.String()))
		if err == nil {
			allWorkspaces = append(allWorkspaces, ws)
		}
	}

	apiWorkspaces, err := convertWorkspaces(r.Context(), api.Database, allWorkspaces)
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("convert workspaces: %s", err),
		})
		return
	}
	httpapi.Write(rw, http.StatusOK, apiWorkspaces)
}

func (api *api) workspacesByOwner(rw http.ResponseWriter, r *http.Request) {
	owner := httpmw.UserParam(r)
	roles := httpmw.UserRoles(r)
	workspaces, err := api.Database.GetWorkspacesByOwnerID(r.Context(), database.GetWorkspacesByOwnerIDParams{
		OwnerID: owner.ID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("get workspaces: %s", err),
		})
		return
	}

	allowedWorkspaces := make([]database.Workspace, 0)
	for _, ws := range workspaces {
		ws := ws
		err = api.Authorizer.ByRoleName(r.Context(), roles.ID.String(), roles.Roles, rbac.ActionRead,
			rbac.ResourceWorkspace.InOrg(ws.OrganizationID).WithOwner(ws.OwnerID.String()).WithID(ws.ID.String()))
		if err == nil {
			allowedWorkspaces = append(allowedWorkspaces, ws)
		}
	}

	apiWorkspaces, err := convertWorkspaces(r.Context(), api.Database, allowedWorkspaces)
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("convert workspaces: %s", err),
		})
		return
	}
	httpapi.Write(rw, http.StatusOK, apiWorkspaces)
}

func (api *api) workspaceByOwnerAndName(rw http.ResponseWriter, r *http.Request) {
	owner := httpmw.UserParam(r)
	organization := httpmw.OrganizationParam(r)
	workspaceName := chi.URLParam(r, "workspace")

	workspace, err := api.Database.GetWorkspaceByOwnerIDAndName(r.Context(), database.GetWorkspaceByOwnerIDAndNameParams{
		OwnerID: owner.ID,
		Name:    workspaceName,
	})
	if errors.Is(err, sql.ErrNoRows) {
		// Do not leak information if the workspace exists or not
		httpapi.Forbidden(rw)
		return
	}
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("get workspace by name: %s", err),
		})
		return
	}

	if workspace.OrganizationID != organization.ID {
		httpapi.Write(rw, http.StatusUnauthorized, httpapi.Response{
			Message: fmt.Sprintf("workspace is not owned by organization %q", organization.Name),
		})
		return
	}

	if !api.Authorize(rw, r, rbac.ActionRead,
		rbac.ResourceWorkspace.InOrg(workspace.OrganizationID).WithOwner(workspace.OwnerID.String()).WithID(workspace.ID.String())) {
		return
	}

	build, err := api.Database.GetWorkspaceBuildByWorkspaceIDWithoutAfter(r.Context(), workspace.ID)
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("get workspace build: %s", err),
		})
		return
	}
	job, err := api.Database.GetProvisionerJobByID(r.Context(), build.JobID)
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("get provisioner job: %s", err),
		})
		return
	}
	template, err := api.Database.GetTemplateByID(r.Context(), workspace.TemplateID)
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("get template: %s", err),
		})
		return
	}

	httpapi.Write(rw, http.StatusOK, convertWorkspace(workspace,
		convertWorkspaceBuild(build, convertProvisionerJob(job)), template, owner))
}

// Create a new workspace for the currently authenticated user.
func (api *api) postWorkspacesByOrganization(rw http.ResponseWriter, r *http.Request) {
	var createWorkspace codersdk.CreateWorkspaceRequest
	if !httpapi.Read(rw, r, &createWorkspace) {
		return
	}
	apiKey := httpmw.APIKey(r)
	template, err := api.Database.GetTemplateByID(r.Context(), createWorkspace.TemplateID)
	if errors.Is(err, sql.ErrNoRows) {
		httpapi.Write(rw, http.StatusBadRequest, httpapi.Response{
			Message: fmt.Sprintf("template %q doesn't exist", createWorkspace.TemplateID.String()),
			Errors: []httpapi.Error{{
				Field:  "template_id",
				Detail: "template not found",
			}},
		})
		return
	}
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("get template: %s", err),
		})
		return
	}
	organization := httpmw.OrganizationParam(r)
	if organization.ID != template.OrganizationID {
		httpapi.Write(rw, http.StatusUnauthorized, httpapi.Response{
			Message: fmt.Sprintf("template is not in organization %q", organization.Name),
		})
		return
	}
	_, err = api.Database.GetOrganizationMemberByUserID(r.Context(), database.GetOrganizationMemberByUserIDParams{
		OrganizationID: template.OrganizationID,
		UserID:         apiKey.UserID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		httpapi.Write(rw, http.StatusUnauthorized, httpapi.Response{
			Message: "you aren't allowed to access templates in that organization",
		})
		return
	}
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("get organization member: %s", err),
		})
		return
	}

	workspace, err := api.Database.GetWorkspaceByOwnerIDAndName(r.Context(), database.GetWorkspaceByOwnerIDAndNameParams{
		OwnerID: apiKey.UserID,
		Name:    createWorkspace.Name,
	})
	if err == nil {
		// If the workspace already exists, don't allow creation.
		template, err := api.Database.GetTemplateByID(r.Context(), workspace.TemplateID)
		if err != nil {
			httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
				Message: fmt.Sprintf("find template for conflicting workspace name %q: %s", createWorkspace.Name, err),
			})
			return
		}
		// The template is fetched for clarity to the user on where the conflicting name may be.
		httpapi.Write(rw, http.StatusConflict, httpapi.Response{
			Message: fmt.Sprintf("workspace %q already exists in the %q template", createWorkspace.Name, template.Name),
			Errors: []httpapi.Error{{
				Field:  "name",
				Detail: "this value is already in use and should be unique",
			}},
		})
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("get workspace by name: %s", err.Error()),
		})
		return
	}

	templateVersion, err := api.Database.GetTemplateVersionByID(r.Context(), template.ActiveVersionID)
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("get template version: %s", err),
		})
		return
	}
	templateVersionJob, err := api.Database.GetProvisionerJobByID(r.Context(), templateVersion.JobID)
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("get template version job: %s", err),
		})
		return
	}
	templateVersionJobStatus := convertProvisionerJob(templateVersionJob).Status
	switch templateVersionJobStatus {
	case codersdk.ProvisionerJobPending, codersdk.ProvisionerJobRunning:
		httpapi.Write(rw, http.StatusNotAcceptable, httpapi.Response{
			Message: fmt.Sprintf("The provided template version is %s. Wait for it to complete importing!", templateVersionJobStatus),
		})
		return
	case codersdk.ProvisionerJobFailed:
		httpapi.Write(rw, http.StatusPreconditionFailed, httpapi.Response{
			Message: fmt.Sprintf("The provided template version %q has failed to import. You cannot create workspaces using it!", templateVersion.Name),
		})
		return
	case codersdk.ProvisionerJobCanceled:
		httpapi.Write(rw, http.StatusPreconditionFailed, httpapi.Response{
			Message: "The provided template version was canceled during import. You cannot create workspaces using it!",
		})
		return
	}

	var provisionerJob database.ProvisionerJob
	var workspaceBuild database.WorkspaceBuild
	err = api.Database.InTx(func(db database.Store) error {
		workspaceBuildID := uuid.New()
		// Workspaces are created without any versions.
		workspace, err = db.InsertWorkspace(r.Context(), database.InsertWorkspaceParams{
			ID:             uuid.New(),
			CreatedAt:      database.Now(),
			UpdatedAt:      database.Now(),
			OwnerID:        apiKey.UserID,
			OrganizationID: template.OrganizationID,
			TemplateID:     template.ID,
			Name:           createWorkspace.Name,
		})
		if err != nil {
			return xerrors.Errorf("insert workspace: %w", err)
		}
		for _, parameterValue := range createWorkspace.ParameterValues {
			_, err = db.InsertParameterValue(r.Context(), database.InsertParameterValueParams{
				ID:                uuid.New(),
				Name:              parameterValue.Name,
				CreatedAt:         database.Now(),
				UpdatedAt:         database.Now(),
				Scope:             database.ParameterScopeWorkspace,
				ScopeID:           workspace.ID,
				SourceScheme:      parameterValue.SourceScheme,
				SourceValue:       parameterValue.SourceValue,
				DestinationScheme: parameterValue.DestinationScheme,
			})
			if err != nil {
				return xerrors.Errorf("insert parameter value: %w", err)
			}
		}

		input, err := json.Marshal(workspaceProvisionJob{
			WorkspaceBuildID: workspaceBuildID,
		})
		if err != nil {
			return xerrors.Errorf("marshal provision job: %w", err)
		}
		provisionerJob, err = db.InsertProvisionerJob(r.Context(), database.InsertProvisionerJobParams{
			ID:             uuid.New(),
			CreatedAt:      database.Now(),
			UpdatedAt:      database.Now(),
			InitiatorID:    apiKey.UserID,
			OrganizationID: template.OrganizationID,
			Provisioner:    template.Provisioner,
			Type:           database.ProvisionerJobTypeWorkspaceBuild,
			StorageMethod:  templateVersionJob.StorageMethod,
			StorageSource:  templateVersionJob.StorageSource,
			Input:          input,
		})
		if err != nil {
			return xerrors.Errorf("insert provisioner job: %w", err)
		}
		workspaceBuild, err = db.InsertWorkspaceBuild(r.Context(), database.InsertWorkspaceBuildParams{
			ID:                workspaceBuildID,
			CreatedAt:         database.Now(),
			UpdatedAt:         database.Now(),
			WorkspaceID:       workspace.ID,
			TemplateVersionID: templateVersion.ID,
			Name:              namesgenerator.GetRandomName(1),
			InitiatorID:       apiKey.UserID,
			Transition:        database.WorkspaceTransitionStart,
			JobID:             provisionerJob.ID,
		})
		if err != nil {
			return xerrors.Errorf("insert workspace build: %w", err)
		}
		return nil
	})
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("create workspace: %s", err),
		})
		return
	}
	user, err := api.Database.GetUserByID(r.Context(), apiKey.UserID)
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("get user: %s", err),
		})
		return
	}

	httpapi.Write(rw, http.StatusCreated, convertWorkspace(workspace,
		convertWorkspaceBuild(workspaceBuild, convertProvisionerJob(templateVersionJob)), template, user))
}

func (api *api) putWorkspaceAutostart(rw http.ResponseWriter, r *http.Request) {
	var req codersdk.UpdateWorkspaceAutostartRequest
	if !httpapi.Read(rw, r, &req) {
		return
	}

	var dbSched sql.NullString
	if req.Schedule != "" {
		validSched, err := schedule.Weekly(req.Schedule)
		if err != nil {
			httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
				Message: fmt.Sprintf("invalid autostart schedule: %s", err),
			})
			return
		}
		dbSched.String = validSched.String()
		dbSched.Valid = true
	}

	workspace := httpmw.WorkspaceParam(r)
	err := api.Database.UpdateWorkspaceAutostart(r.Context(), database.UpdateWorkspaceAutostartParams{
		ID:                workspace.ID,
		AutostartSchedule: dbSched,
	})
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("update workspace autostart schedule: %s", err),
		})
		return
	}
}

func (api *api) putWorkspaceAutostop(rw http.ResponseWriter, r *http.Request) {
	var req codersdk.UpdateWorkspaceAutostopRequest
	if !httpapi.Read(rw, r, &req) {
		return
	}

	var dbSched sql.NullString
	if req.Schedule != "" {
		validSched, err := schedule.Weekly(req.Schedule)
		if err != nil {
			httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
				Message: fmt.Sprintf("invalid autostop schedule: %s", err),
			})
			return
		}
		dbSched.String = validSched.String()
		dbSched.Valid = true
	}

	workspace := httpmw.WorkspaceParam(r)
	err := api.Database.UpdateWorkspaceAutostop(r.Context(), database.UpdateWorkspaceAutostopParams{
		ID:               workspace.ID,
		AutostopSchedule: dbSched,
	})
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("update workspace autostop schedule: %s", err),
		})
		return
	}
}

func convertWorkspaces(ctx context.Context, db database.Store, workspaces []database.Workspace) ([]codersdk.Workspace, error) {
	workspaceIDs := make([]uuid.UUID, 0, len(workspaces))
	templateIDs := make([]uuid.UUID, 0, len(workspaces))
	ownerIDs := make([]uuid.UUID, 0, len(workspaces))
	for _, workspace := range workspaces {
		workspaceIDs = append(workspaceIDs, workspace.ID)
		templateIDs = append(templateIDs, workspace.TemplateID)
		ownerIDs = append(ownerIDs, workspace.OwnerID)
	}
	workspaceBuilds, err := db.GetWorkspaceBuildsByWorkspaceIDsWithoutAfter(ctx, workspaceIDs)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		return nil, xerrors.Errorf("get workspace builds: %w", err)
	}
	templates, err := db.GetTemplatesByIDs(ctx, templateIDs)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		return nil, xerrors.Errorf("get templates: %w", err)
	}
	users, err := db.GetUsersByIDs(ctx, ownerIDs)
	if err != nil {
		return nil, xerrors.Errorf("get users: %w", err)
	}
	jobIDs := make([]uuid.UUID, 0, len(workspaceBuilds))
	for _, build := range workspaceBuilds {
		jobIDs = append(jobIDs, build.JobID)
	}
	jobs, err := db.GetProvisionerJobsByIDs(ctx, jobIDs)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		return nil, xerrors.Errorf("get provisioner jobs: %w", err)
	}

	buildByWorkspaceID := map[uuid.UUID]database.WorkspaceBuild{}
	for _, workspaceBuild := range workspaceBuilds {
		buildByWorkspaceID[workspaceBuild.WorkspaceID] = workspaceBuild
	}
	templateByID := map[uuid.UUID]database.Template{}
	for _, template := range templates {
		templateByID[template.ID] = template
	}
	userByID := map[uuid.UUID]database.User{}
	for _, user := range users {
		userByID[user.ID] = user
	}
	jobByID := map[uuid.UUID]database.ProvisionerJob{}
	for _, job := range jobs {
		jobByID[job.ID] = job
	}
	apiWorkspaces := make([]codersdk.Workspace, 0, len(workspaces))
	for _, workspace := range workspaces {
		build, exists := buildByWorkspaceID[workspace.ID]
		if !exists {
			return nil, xerrors.Errorf("build not found for workspace %q", workspace.Name)
		}
		template, exists := templateByID[workspace.TemplateID]
		if !exists {
			return nil, xerrors.Errorf("template not found for workspace %q", workspace.Name)
		}
		job, exists := jobByID[build.JobID]
		if !exists {
			return nil, xerrors.Errorf("build job not found for workspace: %w", err)
		}
		user, exists := userByID[workspace.OwnerID]
		if !exists {
			return nil, xerrors.Errorf("owner not found for workspace: %q", workspace.Name)
		}
		apiWorkspaces = append(apiWorkspaces,
			convertWorkspace(workspace, convertWorkspaceBuild(build, convertProvisionerJob(job)), template, user))
	}
	return apiWorkspaces, nil
}

func convertWorkspace(workspace database.Workspace, workspaceBuild codersdk.WorkspaceBuild, template database.Template, owner database.User) codersdk.Workspace {
	return codersdk.Workspace{
		ID:                workspace.ID,
		CreatedAt:         workspace.CreatedAt,
		UpdatedAt:         workspace.UpdatedAt,
		OwnerID:           workspace.OwnerID,
		OwnerName:         owner.Username,
		TemplateID:        workspace.TemplateID,
		LatestBuild:       workspaceBuild,
		TemplateName:      template.Name,
		Outdated:          workspaceBuild.TemplateVersionID.String() != template.ActiveVersionID.String(),
		Name:              workspace.Name,
		AutostartSchedule: workspace.AutostartSchedule.String,
		AutostopSchedule:  workspace.AutostopSchedule.String,
	}
}
