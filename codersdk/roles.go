package codersdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/coder/coder/coderd/rbac"
	"github.com/google/uuid"
)

type Role struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

// ListSiteRoles lists all available site wide roles.
// This is not user specific.
func (c *Client) ListSiteRoles(ctx context.Context) ([]Role, error) {
	res, err := c.request(ctx, http.MethodGet, "/api/v2/users/roles", nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, readBodyAsError(res)
	}
	var roles []Role
	return roles, json.NewDecoder(res.Body).Decode(&roles)
}

// ListOrganizationRoles lists all available roles for a given organization.
// This is not user specific.
func (c *Client) ListOrganizationRoles(ctx context.Context, org uuid.UUID) ([]Role, error) {
	res, err := c.request(ctx, http.MethodGet, fmt.Sprintf("/api/v2/organizations/%s/members/roles/", org.String()), nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, readBodyAsError(res)
	}
	var roles []Role
	return roles, json.NewDecoder(res.Body).Decode(&roles)
}

func RolesFromName(roleNames []string) []Role {
	roles := make([]Role, 0, len(roleNames))
	for _, roleName := range roleNames {
		role, err := rbac.RoleByName(roleName)
		if err != nil {
			continue
		}
		roles = append(roles, Role{
			Name:        role.Name,
			DisplayName: role.DisplayName,
		})
	}
	return roles
}
