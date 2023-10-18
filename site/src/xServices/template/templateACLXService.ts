import { updateTemplateACL } from "api/api";
import {
  TemplateACL,
  TemplateGroup,
  TemplateRole,
  TemplateUser,
} from "api/typesGenerated";
import { displaySuccess } from "components/GlobalSnackbar/utils";
import { assign, createMachine } from "xstate";

export const templateACLMachine = createMachine(
  {
    schema: {
      context: {} as {
        templateId: string;
        templateACL?: TemplateACL;
        // Group
        groupToBeAdded?: TemplateGroup;
        groupToBeUpdated?: TemplateGroup;
        addGroupCallback?: () => void;
      },
      services: {} as {
        loadTemplateACL: {
          data: TemplateACL;
        };
        // User
        // Group
        addGroup: {
          data: unknown;
        };
        updateGroup: {
          data: unknown;
        };
      },
      events: {} as  // User
        | {
            type: "LOAD";
            data: TemplateACL;
          }
        | {
            type: "REMOVE_USER";
            user: TemplateUser;
          }
        // Group
        | {
            type: "ADD_GROUP";
            group: TemplateGroup;
            role: TemplateRole;
            onDone: () => void;
          }
        | {
            type: "UPDATE_GROUP_ROLE";
            group: TemplateGroup;
            role: TemplateRole;
          }
        | {
            type: "REMOVE_GROUP";
            group: TemplateGroup;
          },
    },
    tsTypes: {} as import("./templateACLXService.typegen").Typegen0,
    id: "templateACL",
    initial: "loading",
    states: {
      loading: {
        on: {
          LOAD: {
            actions: ["assignTemplateACL"],
            target: "idle",
          },
        },
      },
      idle: {
        on: {
          // User
          REMOVE_USER: {
            target: "removingUser",
            actions: ["removeUserFromTemplateACL"],
          },
          // Group
          ADD_GROUP: {
            target: "addingGroup",
            actions: ["assignGroupToBeAdded"],
          },
          UPDATE_GROUP_ROLE: {
            target: "updatingGroup",
            actions: ["assignGroupToBeUpdated"],
          },
          REMOVE_GROUP: {
            target: "removingGroup",
            actions: ["removeGroupFromTemplateACL"],
          },
        },
      },
      // User
      removingUser: {
        invoke: {
          src: "removeUser",
          onDone: {
            target: "idle",
            actions: ["displayRemoveUserSuccessMessage"],
          },
        },
      },
      // Group
      addingGroup: {
        invoke: {
          src: "addGroup",
          onDone: {
            target: "idle",
            actions: ["addGroupToTemplateACL", "runAddGroupCallback"],
          },
        },
      },
      updatingGroup: {
        invoke: {
          src: "updateGroup",
          onDone: {
            target: "idle",
            actions: [
              "updateGroupOnTemplateACL",
              "clearGroupToBeUpdated",
              "displayUpdateGroupSuccessMessage",
            ],
          },
        },
      },
      removingGroup: {
        invoke: {
          src: "removeGroup",
          onDone: {
            target: "idle",
            actions: ["displayRemoveGroupSuccessMessage"],
          },
        },
      },
    },
  },
  {
    services: {
      // User
      removeUser: ({ templateId }, { user }) =>
        updateTemplateACL(templateId, {
          user_perms: {
            [user.id]: "",
          },
        }),
      // Group
      addGroup: ({ templateId }, { group, role }) =>
        updateTemplateACL(templateId, {
          group_perms: {
            [group.id]: role,
          },
        }),
      updateGroup: ({ templateId }, { group, role }) =>
        updateTemplateACL(templateId, {
          group_perms: {
            [group.id]: role,
          },
        }),
      removeGroup: ({ templateId }, { group }) =>
        updateTemplateACL(templateId, {
          group_perms: {
            [group.id]: "",
          },
        }),
    },
    actions: {
      assignTemplateACL: assign({
        templateACL: (_, { data }) => data,
      }),
      // User
      removeUserFromTemplateACL: assign({
        templateACL: ({ templateACL }, { user }) => {
          if (!templateACL) {
            throw new Error("Template ACL is not loaded yet");
          }
          return {
            ...templateACL,
            users: templateACL.users.filter((oldTemplateUser) => {
              return oldTemplateUser.id !== user.id;
            }),
          };
        },
      }),
      displayRemoveUserSuccessMessage: () => {
        displaySuccess("User removed successfully!");
      },
      // Group
      assignGroupToBeAdded: assign({
        groupToBeAdded: (_, { group, role }) => ({ ...group, role }),
        addGroupCallback: (_, { onDone }) => onDone,
      }),
      addGroupToTemplateACL: assign({
        templateACL: ({ templateACL, groupToBeAdded }) => {
          if (!groupToBeAdded) {
            throw new Error("No group to be added");
          }
          if (!templateACL) {
            throw new Error("Template ACL is not loaded yet");
          }
          return {
            ...templateACL,
            group: [...templateACL.group, groupToBeAdded],
          };
        },
      }),
      runAddGroupCallback: ({ addGroupCallback }) => {
        if (addGroupCallback) {
          addGroupCallback();
        }
      },
      assignGroupToBeUpdated: assign({
        groupToBeUpdated: (_, { group, role }) => ({ ...group, role }),
      }),
      updateGroupOnTemplateACL: assign({
        templateACL: ({ templateACL, groupToBeUpdated }) => {
          if (!groupToBeUpdated) {
            throw new Error("No group to be added");
          }
          if (!templateACL) {
            throw new Error("Template ACL is not loaded yet");
          }
          return {
            ...templateACL,
            group: templateACL.group.map((oldTemplateGroup) => {
              return oldTemplateGroup.id === groupToBeUpdated.id
                ? groupToBeUpdated
                : oldTemplateGroup;
            }),
          };
        },
      }),
      clearGroupToBeUpdated: assign({
        groupToBeUpdated: (_) => undefined,
      }),
      displayUpdateGroupSuccessMessage: () => {
        displaySuccess("Group role update successfully!");
      },
      removeGroupFromTemplateACL: assign({
        templateACL: ({ templateACL }, { group }) => {
          if (!templateACL) {
            throw new Error("Template ACL is not loaded yet");
          }
          return {
            ...templateACL,
            group: templateACL.group.filter((oldTemplateGroup) => {
              return oldTemplateGroup.id !== group.id;
            }),
          };
        },
      }),
      displayRemoveGroupSuccessMessage: () => {
        displaySuccess("Group removed successfully!");
      },
    },
  },
);
