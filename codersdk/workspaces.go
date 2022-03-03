package codersdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/coder/coder/coderd"
)

// Workspaces returns all workspaces the authenticated session has access to.
// If owner is specified, all workspaces for an organization will be returned.
// If owner is empty, all workspaces the caller has access to will be returned.
func (c *Client) Workspaces(ctx context.Context, user string) ([]coderd.Workspace, error) {
	route := "/api/v2/workspaces"
	if user != "" {
		route += fmt.Sprintf("/%s", user)
	}
	res, err := c.request(ctx, http.MethodGet, route, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, readBodyAsError(res)
	}
	var workspaces []coderd.Workspace
	return workspaces, json.NewDecoder(res.Body).Decode(&workspaces)
}

// WorkspacesByProject lists all workspaces for a specific project.
func (c *Client) WorkspacesByProject(ctx context.Context, organization, project string) ([]coderd.Workspace, error) {
	res, err := c.request(ctx, http.MethodGet, fmt.Sprintf("/api/v2/projects/%s/%s/workspaces", organization, project), nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, readBodyAsError(res)
	}
	var workspaces []coderd.Workspace
	return workspaces, json.NewDecoder(res.Body).Decode(&workspaces)
}

// Workspace returns a single workspace by owner and name.
func (c *Client) Workspace(ctx context.Context, owner, name string) (coderd.Workspace, error) {
	if owner == "" {
		owner = "me"
	}
	res, err := c.request(ctx, http.MethodGet, fmt.Sprintf("/api/v2/workspaces/%s/%s", owner, name), nil)
	if err != nil {
		return coderd.Workspace{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return coderd.Workspace{}, readBodyAsError(res)
	}
	var workspace coderd.Workspace
	return workspace, json.NewDecoder(res.Body).Decode(&workspace)
}

// ListWorkspaceHistory returns historical data for workspace builds.
func (c *Client) ListWorkspaceHistory(ctx context.Context, owner, workspace string) ([]coderd.WorkspaceHistory, error) {
	if owner == "" {
		owner = "me"
	}
	res, err := c.request(ctx, http.MethodGet, fmt.Sprintf("/api/v2/workspaces/%s/%s/version", owner, workspace), nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, readBodyAsError(res)
	}
	var workspaceHistory []coderd.WorkspaceHistory
	return workspaceHistory, json.NewDecoder(res.Body).Decode(&workspaceHistory)
}

// WorkspaceHistory returns a single workspace history for a workspace.
// If history is "", the latest version is returned.
func (c *Client) WorkspaceHistory(ctx context.Context, owner, workspace, history string) (coderd.WorkspaceHistory, error) {
	if owner == "" {
		owner = "me"
	}
	if history == "" {
		history = "latest"
	}
	res, err := c.request(ctx, http.MethodGet, fmt.Sprintf("/api/v2/workspaces/%s/%s/version/%s", owner, workspace, history), nil)
	if err != nil {
		return coderd.WorkspaceHistory{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return coderd.WorkspaceHistory{}, readBodyAsError(res)
	}
	var workspaceHistory coderd.WorkspaceHistory
	return workspaceHistory, json.NewDecoder(res.Body).Decode(&workspaceHistory)
}

// CreateWorkspace creates a new workspace for the project specified.
func (c *Client) CreateWorkspace(ctx context.Context, user string, request coderd.CreateWorkspaceRequest) (coderd.Workspace, error) {
	if user == "" {
		user = "me"
	}
	res, err := c.request(ctx, http.MethodPost, fmt.Sprintf("/api/v2/workspaces/%s", user), request)
	if err != nil {
		return coderd.Workspace{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		return coderd.Workspace{}, readBodyAsError(res)
	}
	var workspace coderd.Workspace
	return workspace, json.NewDecoder(res.Body).Decode(&workspace)
}

// CreateWorkspaceHistory queues a new build to occur for a workspace.
func (c *Client) CreateWorkspaceHistory(ctx context.Context, owner, workspace string, request coderd.CreateWorkspaceHistoryRequest) (coderd.WorkspaceHistory, error) {
	if owner == "" {
		owner = "me"
	}
	res, err := c.request(ctx, http.MethodPost, fmt.Sprintf("/api/v2/workspaces/%s/%s/version", owner, workspace), request)
	if err != nil {
		return coderd.WorkspaceHistory{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		return coderd.WorkspaceHistory{}, readBodyAsError(res)
	}
	var workspaceHistory coderd.WorkspaceHistory
	return workspaceHistory, json.NewDecoder(res.Body).Decode(&workspaceHistory)
}

func (c *Client) WorkspaceProvisionJob(ctx context.Context, organization string, job uuid.UUID) (coderd.ProvisionerJob, error) {
	res, err := c.request(ctx, http.MethodGet, fmt.Sprintf("/api/v2/workspaceprovision/%s/%s", organization, job), nil)
	if err != nil {
		return coderd.ProvisionerJob{}, nil
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return coderd.ProvisionerJob{}, readBodyAsError(res)
	}
	var resp coderd.ProvisionerJob
	return resp, json.NewDecoder(res.Body).Decode(&resp)
}

// WorkspaceProvisionJobLogsBefore returns logs that occurred before a specific time.
func (c *Client) WorkspaceProvisionJobLogsBefore(ctx context.Context, organization string, job uuid.UUID, before time.Time) ([]coderd.ProvisionerJobLog, error) {
	return c.provisionerJobLogsBefore(ctx, "workspaceprovision", organization, job, before)
}

// WorkspaceProvisionJobLogsAfter streams logs for a workspace provision operation that occurred after a specific time.
func (c *Client) WorkspaceProvisionJobLogsAfter(ctx context.Context, organization string, job uuid.UUID, after time.Time) (<-chan coderd.ProvisionerJobLog, error) {
	return c.provisionerJobLogsAfter(ctx, "workspaceprovision", organization, job, after)
}

func (c *Client) WorkspaceProvisionJobResources(ctx context.Context, organization string, job uuid.UUID) ([]coderd.ProvisionerJobResource, error) {
	res, err := c.request(ctx, http.MethodGet, fmt.Sprintf("/api/v2/workspaceprovision/%s/%s/resources", organization, job), nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, readBodyAsError(res)
	}
	var resources []coderd.ProvisionerJobResource
	return resources, json.NewDecoder(res.Body).Decode(&resources)
}
