package coderd

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/db2sdk"
	"github.com/coder/coder/v2/coderd/httpapi"
	"github.com/coder/coder/v2/coderd/rbac/rolestore"
	"github.com/coder/coder/v2/codersdk"
)

type enterpriseCustomRoleHandler struct {
	Enabled bool
}

func (h enterpriseCustomRoleHandler) PatchOrganizationRole(ctx context.Context, db database.Store, rw http.ResponseWriter, orgID uuid.UUID, role codersdk.Role) (codersdk.Role, bool) {
	if !h.Enabled {
		httpapi.Write(ctx, rw, http.StatusForbidden, codersdk.Response{
			Message: "Custom roles is not enabled",
		})
		return codersdk.Role{}, false
	}

	// Only organization permissions are allowed to be granted
	if len(role.SitePermissions) > 0 {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message: "Invalid request, not allowed to assign site wide permissions for an organization role.",
			Detail:  "organization scoped roles may not contain site wide permissions",
		})
		return codersdk.Role{}, false
	}

	if len(role.UserPermissions) > 0 {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message: "Invalid request, not allowed to assign user permissions for an organization role.",
			Detail:  "organization scoped roles may not contain user permissions",
		})
		return codersdk.Role{}, false
	}

	if len(role.OrganizationPermissions) > 1 {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message: "Invalid request, Only 1 organization can be assigned permissions",
			Detail:  "roles can only contain 1 organization",
		})
		return codersdk.Role{}, false
	}

	if len(role.OrganizationPermissions) == 1 {
		_, exists := role.OrganizationPermissions[orgID.String()]
		if !exists {
			httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
				Message: fmt.Sprintf("Invalid request, expected permissions for only the organization %q", orgID.String()),
				Detail:  fmt.Sprintf("only org id %s allowed", orgID.String()),
			})
			return codersdk.Role{}, false
		}
	}

	// Make sure all permissions inputted are valid according to our policy.
	rbacRole := db2sdk.RoleToRBAC(role)
	args, err := rolestore.ConvertRoleToDB(rbacRole)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message: "Invalid request",
			Detail:  err.Error(),
		})
		return codersdk.Role{}, false
	}

	inserted, err := db.UpsertCustomRole(ctx, database.UpsertCustomRoleParams{
		Name:        args.Name,
		DisplayName: args.DisplayName,
		OrganizationID: uuid.NullUUID{
			UUID:  orgID,
			Valid: true,
		},
		SitePermissions: args.SitePermissions,
		OrgPermissions:  args.OrgPermissions,
		UserPermissions: args.UserPermissions,
	})
	if httpapi.Is404Error(err) {
		httpapi.ResourceNotFound(rw)
		return codersdk.Role{}, false
	}
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message: "Failed to update role permissions",
			Detail:  err.Error(),
		})
		return codersdk.Role{}, false
	}

	convertedInsert, err := rolestore.ConvertDBRole(inserted)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Permissions were updated, unable to read them back out of the database.",
			Detail:  err.Error(),
		})
		return codersdk.Role{}, false
	}

	return db2sdk.Role(convertedInsert), true
}
