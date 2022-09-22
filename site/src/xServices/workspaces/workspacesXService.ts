import { ActorRefFrom, assign, createMachine, spawn } from "xstate"
import * as API from "../../api/api"
import { getErrorMessage } from "../../api/errors"
import * as TypesGen from "../../api/typesGenerated"
import { displayError, displayMsg, displaySuccess } from "../../components/GlobalSnackbar/utils"
import { queryToFilter } from "../../util/filters"

/**
 * Workspace item machine
 *
 * It is used to control the state and actions of each workspace in the
 * workspaces page view
 **/
interface WorkspaceItemContext {
  data: TypesGen.Workspace
  updatedTemplate?: TypesGen.Template
}

type WorkspaceItemEvent =
  | {
      type: "UPDATE_VERSION"
    }
  | {
      type: "UPDATE_DATA"
      data: TypesGen.Workspace
    }

export const workspaceItemMachine = createMachine(
  {
    id: "workspaceItemMachine",
    predictableActionArguments: true,
    tsTypes: {} as import("./workspacesXService.typegen").Typegen0,
    schema: {
      context: {} as WorkspaceItemContext,
      events: {} as WorkspaceItemEvent,
      services: {} as {
        getTemplate: {
          data: TypesGen.Template
        }
        startWorkspace: {
          data: TypesGen.WorkspaceBuild
        }
        getWorkspace: {
          data: TypesGen.Workspace
        }
      },
    },
    type: "parallel",

    states: {
      updateVersion: {
        initial: "idle",
        states: {
          idle: {
            on: {
              UPDATE_VERSION: {
                target: "gettingUpdatedTemplate",
                // We can improve the UI by optimistically updating the workspace status
                // to "Queued" so the UI can display the updated state right away and we
                // don't need to display an extra spinner.
                actions: ["assignQueuedStatus", "displayUpdatingVersionMessage"],
              },
              UPDATE_DATA: {
                actions: "assignUpdatedData",
              },
            },
          },
          gettingUpdatedTemplate: {
            invoke: {
              id: "getTemplate",
              src: "getTemplate",
              onDone: {
                actions: "assignUpdatedTemplate",
                target: "restartingWorkspace",
              },
              onError: {
                target: "idle",
                actions: "displayUpdateVersionError",
              },
            },
          },
          restartingWorkspace: {
            invoke: {
              id: "startWorkspace",
              src: "startWorkspace",
              onDone: {
                actions: "assignLatestBuild",
                target: "waitingToBeUpdated",
              },
              onError: {
                target: "idle",
                actions: "displayUpdateVersionError",
              },
            },
          },
          waitingToBeUpdated: {
            after: {
              5000: "gettingUpdatedWorkspaceData",
            },
          },
          gettingUpdatedWorkspaceData: {
            invoke: {
              id: "getWorkspace",
              src: "getWorkspace",
              onDone: [
                {
                  target: "waitingToBeUpdated",
                  cond: "isOutdated",
                  actions: ["assignUpdatedData"],
                },
                {
                  target: "idle",
                  actions: ["assignUpdatedData", "displayUpdatedSuccessMessage"],
                },
              ],
            },
          },
        },
      },
    },
  },
  {
    guards: {
      isOutdated: (_, event) => event.data.outdated,
    },
    services: {
      getTemplate: (context) => API.getTemplate(context.data.template_id),
      startWorkspace: (context) => {
        if (!context.updatedTemplate) {
          throw new Error("Updated template is not loaded.")
        }

        return API.startWorkspace(context.data.id, context.updatedTemplate.active_version_id)
      },
      getWorkspace: (context) => API.getWorkspace(context.data.id),
    },
    actions: {
      assignUpdatedTemplate: assign({
        updatedTemplate: (_, event) => event.data,
      }),
      assignLatestBuild: assign({
        data: (context, event) => {
          return {
            ...context.data,
            latest_build: event.data,
          }
        },
      }),
      displayUpdateVersionError: (_, event) => {
        const message = getErrorMessage(event.data, "Error on update workspace version.")
        displayError(message)
      },
      displayUpdatingVersionMessage: () => {
        displayMsg("Updating workspace...")
      },
      assignQueuedStatus: assign({
        data: (ctx) => {
          return {
            ...ctx.data,
            latest_build: {
              ...ctx.data.latest_build,
              job: {
                ...ctx.data.latest_build.job,
                status: "pending" as TypesGen.ProvisionerJobStatus,
              },
            },
          }
        },
      }),
      displayUpdatedSuccessMessage: () => {
        displaySuccess("Workspace updated successfully.")
      },
      assignUpdatedData: assign({
        data: (_, event) => event.data,
      }),
    },
  },
)

/**
 * Workspaces machine
 *
 * It is used to control the state of the workspace list
 **/

export type WorkspaceItemMachineRef = ActorRefFrom<typeof workspaceItemMachine>

interface WorkspacesContext {
  workspaceRefs?: WorkspaceItemMachineRef[]
  filter: string
  getWorkspacesError?: Error | unknown
}

type WorkspacesEvent =
  | { type: "GET_WORKSPACES"; query?: string }
  | { type: "UPDATE_VERSION"; workspaceId: string }

export const workspacesMachine =
/** @xstate-layout N4IgpgJg5mDOIC5QHcD2AnA1rADgQwGM4BlAFz1LADoZTSBLAOygHUNt8jYBiCVR6kwBuqTNVpssuQnESgcqWPQb85IAB6IAbABYdVAIwB2AwE4ATAFYAzAA5TO+zoA0IAJ6ID1o1VN-Ttra6Olrm5gAMtgC+Ua5oUpwk5JQ0YHRMrOzSXNxg6OgYVDgANhQAZhgAtqmkkhwy8EggCkoqjGqaCMGGJhY2Tk6uHgjWOua+-lqBpuF+lg4xcVmJsGQU1ACuOBAUGXXZYABKYGU8fAJUwqKb2+v7icenai3K9KpNnd3GZlZ2DoPuRBjWwTPxaSxhHQGSwQxYgeL1LhrFLIPDKAAqqEe6DgAAt7g1uOpYMlqHgypR0AAKSzhOkASm4CIOq1JVFRGKxJxxsHxywaz0Ur3eoE6RhMVHBWms5ls1nCOiMplGQ0Q9lBc1Co0cszhzJWyLA3AA4gBRdEAfRYAHlDgBpYgABQAggBhU3EQWtN7tD6ISxGVUIMz6fwBcJacIGMILWLw-lI0ncACqjoAIs70aaLQA1U2HYgASWtADkvcLfaL-dGqDpTFoDHorOKjCEg5YtKYNVK7NDrNYYnHGKgILImvqGobLhBimBy20Op5IlQ5fNrAFwUZwt5zEHzDMNUZO6M7EEtHqE0l1jUGMwCVx5z7FwhzNLa62tA2xgrbBFLEGDF0DUZjpOsoyMSwLwSSc2S2HZb0yaCiEeRp5CFBc-RfPQegMewwk7aUjHMawgx0BVuwMRt+0A8woMRK8UTRUhMWxPF7zHNDvRFDREAsEEGyIpVwghIwTwAqMVzDeZP0VSxozollDUfbjOn7dtgLDTSB0HIA */
createMachine(
  {
  tsTypes: {} as import("./workspacesXService.typegen").Typegen1,
  schema: {
    context: {} as WorkspacesContext,
    events: {} as WorkspacesEvent,
    services: {} as {
      getWorkspaces: {
        data: TypesGen.Workspace[]
      }
      updateWorkspaceRefs: {
        data: {
          refsToKeep: WorkspaceItemMachineRef[]
          newWorkspaces: TypesGen.Workspace[]
        }
      }
    },
  },
  predictableActionArguments: true,
  id: "workspacesState",
  on: {
    GET_WORKSPACES: {
      actions: "assignFilter",
      target: ".gettingWorkspaces",
      internal: false,
    },
    UPDATE_VERSION: {
      actions: "triggerUpdateVersion",
    },
  },
  initial: "gettingWorkspaces",
  states: {
    gettingWorkspaces: {
      entry: "clearGetWorkspacesError",
      invoke: {
        src: "getWorkspaces",
        id: "getWorkspaces",
        onDone: [
          {
            actions: "assignWorkspaceRefs",
            cond: "isEmpty",
            target: "waitToRefreshWorkspaces",
          },
          {
            target: "updatingWorkspaceRefs",
          },
        ],
        onError: [
          {
            actions: "assignGetWorkspacesError",
            target: "waitToRefreshWorkspaces",
          },
        ],
      },
    },
    updatingWorkspaceRefs: {
      invoke: {
        src: "updateWorkspaceRefs",
        id: "updateWorkspaceRefs",
        onDone: [
          {
            actions: "assignUpdatedWorkspaceRefs",
            target: "waitToRefreshWorkspaces",
          },
        ],
      },
    },
    waitToRefreshWorkspaces: {
      after: {
        "5000": {
          target: "gettingWorkspaces",
        },
      },
    },
  },
},
  {
    guards: {
      isEmpty: (context) => !context.workspaceRefs,
    },
    actions: {
      assignWorkspaceRefs: assign({
        workspaceRefs: (_, event) =>
          event.data.map((data) => {
            return spawn(workspaceItemMachine.withContext({ data }), data.id)
          }),
      }),
      assignFilter: assign({
        filter: (context, event) => event.query ?? context.filter,
      }),
      assignGetWorkspacesError: assign({
        getWorkspacesError: (_, event) => event.data,
      }),
      clearGetWorkspacesError: (context) => assign({ ...context, getWorkspacesError: undefined }),
      triggerUpdateVersion: (context, event) => {
        const workspaceRef = context.workspaceRefs?.find((ref) => ref.id === event.workspaceId)

        if (!workspaceRef) {
          throw new Error(`No workspace ref found for ${event.workspaceId}.`)
        }

        workspaceRef.send("UPDATE_VERSION")
      },
      assignUpdatedWorkspaceRefs: assign({
        workspaceRefs: (_, event) => {
          const newWorkspaceRefs = event.data.newWorkspaces.map((workspace) =>
            spawn(workspaceItemMachine.withContext({ data: workspace }), workspace.id),
          )
          return event.data.refsToKeep.concat(newWorkspaceRefs)
        },
      }),
    },
    services: {
      getWorkspaces: (context) => API.getWorkspaces(queryToFilter(context.filter)),
      updateWorkspaceRefs: (context, event) => {
        const refsToKeep: WorkspaceItemMachineRef[] = []
        context.workspaceRefs?.forEach((ref) => {
          const matchingWorkspace = event.data.find((workspace) => ref.id === workspace.id)
          if (matchingWorkspace) {
            ref.send({ type: "UPDATE_DATA", data: matchingWorkspace })
            refsToKeep.push(ref)
          } else {
            ref.stop && ref.stop()
          }
        })

        const newWorkspaces = event.data.filter(
          (workspace) => !context.workspaceRefs?.find((ref) => ref.id === workspace.id),
        )

        return Promise.resolve({
          refsToKeep,
          newWorkspaces,
        })
      },
    },
  },
)
