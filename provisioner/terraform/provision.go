package terraform

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/xerrors"

	"cdr.dev/slog"

	"github.com/coder/coder/provisionersdk/proto"
)

// Provision executes `terraform apply`.
func (t *terraform) Provision(request *proto.Provision_Request, stream proto.DRPCProvisioner_ProvisionStream) error {
	ctx := stream.Context()
	statefilePath := filepath.Join(request.Directory, "terraform.tfstate")
	if len(request.State) > 0 {
		err := os.WriteFile(statefilePath, request.State, 0600)
		if err != nil {
			return xerrors.Errorf("write statefile %q: %w", statefilePath, err)
		}
	}

	terraform, err := tfexec.NewTerraform(request.Directory, t.binaryPath)
	if err != nil {
		return xerrors.Errorf("create new terraform executor: %w", err)
	}
	version, _, err := terraform.Version(ctx, false)
	if err != nil {
		return xerrors.Errorf("get terraform version: %w", err)
	}
	if !version.GreaterThanOrEqual(minimumTerraformVersion) {
		return xerrors.Errorf("terraform version %q is too old. required >= %q", version.String(), minimumTerraformVersion.String())
	}

	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			_ = stream.Send(&proto.Provision_Response{
				Type: &proto.Provision_Response_Log{
					Log: &proto.Log{
						Level:  proto.LogLevel_INFO,
						Output: scanner.Text(),
					},
				},
			})
		}
	}()
	terraform.SetStdout(writer)
	t.logger.Debug(ctx, "running initialization")
	err = terraform.Init(ctx)
	if err != nil {
		return xerrors.Errorf("initialize terraform: %w", err)
	}
	t.logger.Debug(ctx, "ran initialization")

	if request.DryRun {
		return t.runTerraformPlan(ctx, terraform, request, stream)
	}
	return t.runTerraformApply(ctx, terraform, request, stream, statefilePath)
}

func (t *terraform) runTerraformPlan(ctx context.Context, terraform *tfexec.Terraform, request *proto.Provision_Request, stream proto.DRPCProvisioner_ProvisionStream) error {
	env := map[string]string{}
	for _, envEntry := range os.Environ() {
		parts := strings.SplitN(envEntry, "=", 2)
		env[parts[0]] = parts[1]
	}
	env["CODER_URL"] = request.Metadata.CoderUrl
	env["CODER_WORKSPACE_TRANSITION"] = strings.ToLower(request.Metadata.WorkspaceTransition.String())
	planfilePath := filepath.Join(request.Directory, "terraform.tfplan")
	options := []tfexec.PlanOption{tfexec.JSON(true), tfexec.Out(planfilePath)}
	for _, param := range request.ParameterValues {
		switch param.DestinationScheme {
		case proto.ParameterDestination_ENVIRONMENT_VARIABLE:
			env[param.Name] = param.Value
		case proto.ParameterDestination_PROVISIONER_VARIABLE:
			options = append(options, tfexec.Var(fmt.Sprintf("%s=%s", param.Name, param.Value)))
		default:
			return xerrors.Errorf("unsupported parameter type %q for %q", param.DestinationScheme, param.Name)
		}
	}
	err := terraform.SetEnv(env)
	if err != nil {
		return xerrors.Errorf("apply environment variables: %w", err)
	}

	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	closeChan := make(chan struct{})
	go func() {
		defer close(closeChan)
		decoder := json.NewDecoder(reader)
		for {
			var log terraformProvisionLog
			err := decoder.Decode(&log)
			if err != nil {
				return
			}

			logLevel, err := convertTerraformLogLevel(log.Level)
			if err != nil {
				// Not a big deal, but we should handle this at some point!
				continue
			}
			_ = stream.Send(&proto.Provision_Response{
				Type: &proto.Provision_Response_Log{
					Log: &proto.Log{
						Level:  logLevel,
						Output: log.Message,
					},
				},
			})

			if log.Diagnostic == nil {
				continue
			}

			// If the diagnostic is provided, let's provide a bit more info!
			logLevel, err = convertTerraformLogLevel(log.Diagnostic.Severity)
			if err != nil {
				continue
			}
			_ = stream.Send(&proto.Provision_Response{
				Type: &proto.Provision_Response_Log{
					Log: &proto.Log{
						Level:  logLevel,
						Output: log.Diagnostic.Detail,
					},
				},
			})
		}
	}()

	terraform.SetStdout(writer)
	t.logger.Debug(ctx, "running plan")
	_, err = terraform.Plan(ctx, options...)
	if err != nil {
		return xerrors.Errorf("plan terraform: %w", err)
	}
	t.logger.Debug(ctx, "ran plan")

	plan, err := terraform.ShowPlanFile(ctx, planfilePath)
	if err != nil {
		return xerrors.Errorf("show terraform plan file: %w", err)
	}
	_ = reader.Close()
	<-closeChan

	// Maps resource dependencies to expression references.
	// This is *required* for a plan, because "DependsOn"
	// does not propagate.
	resourceDependencies := map[string][]string{}
	for _, resource := range plan.Config.RootModule.Resources {
		if resource.Expressions == nil {
			resource.Expressions = map[string]*tfjson.Expression{}
		}
		// Count expression is separated for logical reasons,
		// but it's simpler syntactically for us to combine here.
		if resource.CountExpression != nil {
			resource.Expressions["count"] = resource.CountExpression
		}
		for _, expression := range resource.Expressions {
			dependencies, exists := resourceDependencies[resource.Address]
			if !exists {
				dependencies = []string{}
			}
			dependencies = append(dependencies, expression.References...)
			resourceDependencies[resource.Address] = dependencies
		}
	}

	resources := make([]*proto.Resource, 0)
	agents := map[string]*proto.Agent{}

	// Store all agents inside the maps!
	for _, resource := range plan.Config.RootModule.Resources {
		if resource.Type != "coder_agent" {
			continue
		}
		agent := &proto.Agent{
			Auth: &proto.Agent_Token{},
		}
		if envRaw, has := resource.Expressions["env"]; has {
			env, ok := envRaw.ConstantValue.(map[string]string)
			if !ok {
				return xerrors.Errorf("unexpected type %q for env map", reflect.TypeOf(envRaw.ConstantValue).String())
			}
			agent.Env = env
		}
		if startupScriptRaw, has := resource.Expressions["startup_script"]; has {
			startupScript, ok := startupScriptRaw.ConstantValue.(string)
			if !ok {
				return xerrors.Errorf("unexpected type %q for startup script", reflect.TypeOf(startupScriptRaw.ConstantValue).String())
			}
			agent.StartupScript = startupScript
		}
		if auth, has := resource.Expressions["auth"]; has {
			if len(auth.ExpressionData.NestedBlocks) > 0 {
				block := auth.ExpressionData.NestedBlocks[0]
				authType, has := block["type"]
				if has {
					authTypeValue, valid := authType.ConstantValue.(string)
					if !valid {
						return xerrors.Errorf("unexpected type %q for auth type", reflect.TypeOf(authType.ConstantValue))
					}
					switch authTypeValue {
					case "google-instance-identity":
						instanceID, _ := block["instance_id"].ConstantValue.(string)
						agent.Auth = &proto.Agent_GoogleInstanceIdentity{
							GoogleInstanceIdentity: &proto.GoogleInstanceIdentityAuth{
								InstanceId: instanceID,
							},
						}
					default:
						return xerrors.Errorf("unknown auth type: %q", authTypeValue)
					}
				}
			}
		}

		agents[resource.Address] = agent
	}

	for _, resource := range plan.PlannedValues.RootModule.Resources {
		if resource.Type == "coder_agent" {
			continue
		}
		// The resource address on planned values can include the indexed
		// value like "[0]", but the config doesn't have these, and we don't
		// care which index the resource is.
		resourceAddress := fmt.Sprintf("%s.%s", resource.Type, resource.Name)
		var agent *proto.Agent
		// Associate resources that depend on an agent.
		for _, dependency := range resourceDependencies[resourceAddress] {
			var has bool
			agent, has = agents[dependency]
			if has {
				break
			}
		}
		// Associate resources where the agent depends on it.
		for agentAddress := range agents {
			for _, depend := range resourceDependencies[agentAddress] {
				if depend != resource.Address {
					continue
				}
				agent = agents[agentAddress]
				break
			}
		}

		resources = append(resources, &proto.Resource{
			Name:  resource.Name,
			Type:  resource.Type,
			Agent: agent,
		})
	}

	return stream.Send(&proto.Provision_Response{
		Type: &proto.Provision_Response_Complete{
			Complete: &proto.Provision_Complete{
				Resources: resources,
			},
		},
	})
}

func (t *terraform) runTerraformApply(ctx context.Context, terraform *tfexec.Terraform, request *proto.Provision_Request, stream proto.DRPCProvisioner_ProvisionStream, statefilePath string) error {
	env := os.Environ()
	env = append(env,
		"CODER_URL="+request.Metadata.CoderUrl,
		"CODER_WORKSPACE_TRANSITION="+strings.ToLower(request.Metadata.WorkspaceTransition.String()),
	)
	vars := []string{}
	for _, param := range request.ParameterValues {
		switch param.DestinationScheme {
		case proto.ParameterDestination_ENVIRONMENT_VARIABLE:
			env = append(env, fmt.Sprintf("%s=%s", param.Name, param.Value))
		case proto.ParameterDestination_PROVISIONER_VARIABLE:
			vars = append(vars, fmt.Sprintf("%s=%s", param.Name, param.Value))
		default:
			return xerrors.Errorf("unsupported parameter type %q for %q", param.DestinationScheme, param.Name)
		}
	}

	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	go func() {
		decoder := json.NewDecoder(reader)
		for {
			var log terraformProvisionLog
			err := decoder.Decode(&log)
			if err != nil {
				return
			}

			logLevel, err := convertTerraformLogLevel(log.Level)
			if err != nil {
				// Not a big deal, but we should handle this at some point!
				continue
			}
			_ = stream.Send(&proto.Provision_Response{
				Type: &proto.Provision_Response_Log{
					Log: &proto.Log{
						Level:  logLevel,
						Output: log.Message,
					},
				},
			})

			if log.Diagnostic == nil {
				continue
			}

			// If the diagnostic is provided, let's provide a bit more info!
			logLevel, err = convertTerraformLogLevel(log.Diagnostic.Severity)
			if err != nil {
				continue
			}
			_ = stream.Send(&proto.Provision_Response{
				Type: &proto.Provision_Response_Log{
					Log: &proto.Log{
						Level:  logLevel,
						Output: log.Diagnostic.Detail,
					},
				},
			})
		}
	}()

	t.logger.Debug(ctx, "running apply", slog.F("vars", len(vars)), slog.F("env", len(env)))
	err := runApplyCommand(ctx, t.shutdownCtx, request.Metadata.WorkspaceTransition, terraform.ExecPath(), terraform.WorkingDir(), writer, env, vars)
	if err != nil {
		errorMessage := err.Error()
		// Terraform can fail and apply and still need to store it's state.
		// In this case, we return Complete with an explicit error message.
		statefileContent, err := os.ReadFile(statefilePath)
		if err != nil {
			return xerrors.Errorf("read file %q: %w", statefilePath, err)
		}
		return stream.Send(&proto.Provision_Response{
			Type: &proto.Provision_Response_Complete{
				Complete: &proto.Provision_Complete{
					State: statefileContent,
					Error: errorMessage,
				},
			},
		})
	}
	t.logger.Debug(ctx, "ran apply")

	statefileContent, err := os.ReadFile(statefilePath)
	if err != nil {
		return xerrors.Errorf("read file %q: %w", statefilePath, err)
	}
	state, err := terraform.ShowStateFile(ctx, statefilePath)
	if err != nil {
		return xerrors.Errorf("show state file %q: %w", statefilePath, err)
	}
	resources := make([]*proto.Resource, 0)
	if state.Values != nil {
		type agentAttributes struct {
			ID    string `mapstructure:"id"`
			Token string `mapstructure:"token"`
			Auth  []struct {
				Type       string `mapstructure:"type"`
				InstanceID string `mapstructure:"instance_id"`
			} `mapstructure:"auth"`
			Env           map[string]string `mapstructure:"env"`
			StartupScript string            `mapstructure:"startup_script"`
		}
		agents := map[string]*proto.Agent{}
		agentDepends := map[string][]string{}

		// Store all agents inside the maps!
		for _, resource := range state.Values.RootModule.Resources {
			if resource.Type != "coder_agent" {
				continue
			}
			var attrs agentAttributes
			err = mapstructure.Decode(resource.AttributeValues, &attrs)
			if err != nil {
				return xerrors.Errorf("decode agent attributes: %w", err)
			}
			agent := &proto.Agent{
				Id:            attrs.ID,
				Env:           attrs.Env,
				StartupScript: attrs.StartupScript,
				Auth: &proto.Agent_Token{
					Token: attrs.Token,
				},
			}
			if len(attrs.Auth) > 0 {
				auth := attrs.Auth[0]
				switch auth.Type {
				case "google-instance-identity":
					agent.Auth = &proto.Agent_GoogleInstanceIdentity{
						GoogleInstanceIdentity: &proto.GoogleInstanceIdentityAuth{
							InstanceId: auth.InstanceID,
						},
					}
				default:
					return xerrors.Errorf("unknown auth type: %q", auth.Type)
				}
			}
			resourceKey := strings.Join([]string{resource.Type, resource.Name}, ".")
			agents[resourceKey] = agent
			agentDepends[resourceKey] = resource.DependsOn
		}

		for _, resource := range state.Values.RootModule.Resources {
			if resource.Type == "coder_agent" {
				continue
			}
			var agent *proto.Agent
			// Associate resources that depend on an agent.
			for _, dep := range resource.DependsOn {
				var has bool
				agent, has = agents[dep]
				if has {
					break
				}
			}
			// Associate resources where the agent depends on it.
			for agentKey, dependsOn := range agentDepends {
				for _, depend := range dependsOn {
					if depend != strings.Join([]string{resource.Type, resource.Name}, ".") {
						continue
					}
					agent = agents[agentKey]
					break
				}
			}

			resources = append(resources, &proto.Resource{
				Name:  resource.Name,
				Type:  resource.Type,
				Agent: agent,
			})
		}
	}

	return stream.Send(&proto.Provision_Response{
		Type: &proto.Provision_Response_Complete{
			Complete: &proto.Provision_Complete{
				State:     statefileContent,
				Resources: resources,
			},
		},
	})
}

// This couldn't use terraform-exec, because it doesn't support cancellation, and there didn't appear
// to be a straight-forward way to add it.
func runApplyCommand(ctx, shutdownCtx context.Context, transition proto.WorkspaceTransition, bin, dir string, stdout io.Writer, env, vars []string) error {
	args := []string{
		"apply",
		"-no-color",
		"-auto-approve",
		"-input=false",
		"-json",
		"-refresh=true",
	}
	if transition == proto.WorkspaceTransition_DESTROY {
		args = append(args, "-destroy")
	}
	for _, variable := range vars {
		args = append(args, "-var", variable)
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-shutdownCtx.Done():
			_ = cmd.Process.Signal(os.Kill)
		}
	}()
	cmd.Stdout = stdout
	cmd.Env = env
	cmd.Dir = dir
	return cmd.Run()
}

type terraformProvisionLog struct {
	Level   string `json:"@level"`
	Message string `json:"@message"`

	Diagnostic *terraformProvisionLogDiagnostic `json:"diagnostic"`
}

type terraformProvisionLogDiagnostic struct {
	Severity string `json:"severity"`
	Summary  string `json:"summary"`
	Detail   string `json:"detail"`
}

func convertTerraformLogLevel(logLevel string) (proto.LogLevel, error) {
	switch strings.ToLower(logLevel) {
	case "trace":
		return proto.LogLevel_TRACE, nil
	case "debug":
		return proto.LogLevel_DEBUG, nil
	case "info":
		return proto.LogLevel_INFO, nil
	case "warn":
		return proto.LogLevel_WARN, nil
	case "error":
		return proto.LogLevel_ERROR, nil
	default:
		return proto.LogLevel(0), xerrors.Errorf("invalid log level %q", logLevel)
	}
}
