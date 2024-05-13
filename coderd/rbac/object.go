package rbac

import (
	"github.com/google/uuid"

	"github.com/coder/coder/v2/coderd/rbac/policy"
)

const WildcardSymbol = "*"

// Objecter returns the RBAC object for itself.
type Objecter interface {
	RBACObject() Object
}

// ResourceUserObject is a helper function to create a user object for authz checks.
func ResourceUserObject(userID uuid.UUID) Object {
	return ResourceUser.WithID(userID).WithOwner(userID.String())
}

// Object is used to create objects for authz checks when you have none in
// hand to run the check on.
// An example is if you want to list all workspaces, you can create a Object
// that represents the set of workspaces you are trying to get access too.
// Do not export this type, as it can be created from a resource type constant.
type Object struct {
	// ID is the resource's uuid
	ID    string `json:"id"`
	Owner string `json:"owner"`
	// OrgID specifies which org the object is a part of.
	OrgID string `json:"org_owner"`

	// Type is "workspace", "project", "app", etc
	Type string `json:"type"`

	ACLUserList  map[string][]policy.Action ` json:"acl_user_list"`
	ACLGroupList map[string][]policy.Action ` json:"acl_group_list"`
}

func (z Object) AvailableActions() []policy.Action {
	policy.Action()
}

func (z Object) Equal(b Object) bool {
	if z.ID != b.ID {
		return false
	}
	if z.Owner != b.Owner {
		return false
	}
	if z.OrgID != b.OrgID {
		return false
	}
	if z.Type != b.Type {
		return false
	}

	if !equalACLLists(z.ACLUserList, b.ACLUserList) {
		return false
	}

	if !equalACLLists(z.ACLGroupList, b.ACLGroupList) {
		return false
	}

	return true
}

func equalACLLists(a, b map[string][]policy.Action) bool {
	if len(a) != len(b) {
		return false
	}

	for k, actions := range a {
		if len(actions) != len(b[k]) {
			return false
		}
		for i, a := range actions {
			if a != b[k][i] {
				return false
			}
		}
	}
	return true
}

func (z Object) RBACObject() Object {
	return z
}

// All returns an object matching all resources of the same type.
func (z Object) All() Object {
	return Object{
		Owner:        "",
		OrgID:        "",
		Type:         z.Type,
		ACLUserList:  map[string][]policy.Action{},
		ACLGroupList: map[string][]policy.Action{},
	}
}

func (z Object) WithIDString(id string) Object {
	return Object{
		ID:           id,
		Owner:        z.Owner,
		OrgID:        z.OrgID,
		Type:         z.Type,
		ACLUserList:  z.ACLUserList,
		ACLGroupList: z.ACLGroupList,
	}
}

func (z Object) WithID(id uuid.UUID) Object {
	return Object{
		ID:           id.String(),
		Owner:        z.Owner,
		OrgID:        z.OrgID,
		Type:         z.Type,
		ACLUserList:  z.ACLUserList,
		ACLGroupList: z.ACLGroupList,
	}
}

// InOrg adds an org OwnerID to the resource
func (z Object) InOrg(orgID uuid.UUID) Object {
	return Object{
		ID:           z.ID,
		Owner:        z.Owner,
		OrgID:        orgID.String(),
		Type:         z.Type,
		ACLUserList:  z.ACLUserList,
		ACLGroupList: z.ACLGroupList,
	}
}

// WithOwner adds an OwnerID to the resource
func (z Object) WithOwner(ownerID string) Object {
	return Object{
		ID:           z.ID,
		Owner:        ownerID,
		OrgID:        z.OrgID,
		Type:         z.Type,
		ACLUserList:  z.ACLUserList,
		ACLGroupList: z.ACLGroupList,
	}
}

// WithACLUserList adds an ACL list to a given object
func (z Object) WithACLUserList(acl map[string][]policy.Action) Object {
	return Object{
		ID:           z.ID,
		Owner:        z.Owner,
		OrgID:        z.OrgID,
		Type:         z.Type,
		ACLUserList:  acl,
		ACLGroupList: z.ACLGroupList,
	}
}

func (z Object) WithGroupACL(groups map[string][]policy.Action) Object {
	return Object{
		ID:           z.ID,
		Owner:        z.Owner,
		OrgID:        z.OrgID,
		Type:         z.Type,
		ACLUserList:  z.ACLUserList,
		ACLGroupList: groups,
	}
}
