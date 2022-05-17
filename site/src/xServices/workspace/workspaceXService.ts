import { assign, createMachine, send } from "xstate"
import { pure } from "xstate/lib/actions"
import * as API from "../../api/api"
import * as TypesGen from "../../api/typesGenerated"
import { displayError } from "../../components/GlobalSnackbar/utils"

const latestBuild = (builds: TypesGen.WorkspaceBuild[]) => {
  // Cloning builds to not change the origin object with the sort()
  return [...builds].sort((a, b) => {
    return new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
  })[0]
}

const Language = {
  refreshTemplateError: "Error updating workspace: latest template could not be fetched.",
  buildError: "Workspace action failed.",
}

export interface WorkspaceContext {
  workspace?: TypesGen.Workspace
  template?: TypesGen.Template
  organization?: TypesGen.Organization
  build?: TypesGen.WorkspaceBuild
  getWorkspaceError?: Error | unknown
  getTemplateError?: Error | unknown
  getOrganizationError?: Error | unknown
  // error creating a new WorkspaceBuild
  buildError?: Error | unknown
  // these are separate from getX errors because they don't make the page unusable
  refreshWorkspaceError: Error | unknown
  refreshTemplateError: Error | unknown
  // Builds
  builds?: TypesGen.WorkspaceBuild[]
  getBuildsError?: Error | unknown
  loadMoreBuildsError?: Error | unknown
}

export type WorkspaceEvent =
  | { type: "GET_WORKSPACE"; workspaceId: string }
  | { type: "START" }
  | { type: "STOP" }
  | { type: "RETRY" }
  | { type: "UPDATE" }
  | { type: "LOAD_MORE_BUILDS" }
  | { type: "REFRESH_TIMELINE" }

export const workspaceMachine = createMachine(
  {
    tsTypes: {} as import("./workspaceXService.typegen").Typegen0,
    schema: {
      context: {} as WorkspaceContext,
      events: {} as WorkspaceEvent,
      services: {} as {
        getWorkspace: {
          data: TypesGen.Workspace
        }
        getTemplate: {
          data: TypesGen.Template
        }
        getOrganization: {
          data: TypesGen.Organization
        }
        startWorkspace: {
          data: TypesGen.WorkspaceBuild
        }
        stopWorkspace: {
          data: TypesGen.WorkspaceBuild
        }
        refreshWorkspace: {
          data: TypesGen.Workspace | undefined
        }
        getBuilds: {
          data: TypesGen.WorkspaceBuild[]
        }
        loadMoreBuilds: {
          data: TypesGen.WorkspaceBuild[]
        }
      },
    },
    id: "workspaceState",
    initial: "idle",
    on: {
      GET_WORKSPACE: "gettingWorkspace",
    },
    states: {
      idle: {
        tags: "loading",
      },
      gettingWorkspace: {
        entry: ["clearGetWorkspaceError", "clearContext"],
        invoke: {
          src: "getWorkspace",
          id: "getWorkspace",
          onDone: {
            target: "ready",
            actions: ["assignWorkspace"],
          },
          onError: {
            target: "error",
            actions: "assignGetWorkspaceError",
          },
        },
        tags: "loading",
      },
      ready: {
        type: "parallel",
        states: {
          // We poll the workspace consistently to know if it becomes outdated and to update build status
          pollingWorkspace: {
            initial: "refreshingWorkspace",
            states: {
              refreshingWorkspace: {
                entry: "clearRefreshWorkspaceError",
                invoke: {
                  id: "refreshWorkspace",
                  src: "refreshWorkspace",
                  onDone: { target: "waiting", actions: ["refreshTimeline", "assignWorkspace"] },
                  onError: { target: "waiting", actions: "assignRefreshWorkspaceError" },
                },
              },
              waiting: {
                after: {
                  1000: "refreshingWorkspace",
                },
              },
            },
          },
          breadcrumb: {
            initial: "gettingTemplate",
            states: {
              gettingTemplate: {
                invoke: {
                  src: "getTemplate",
                  id: "getTemplate",
                  onDone: {
                    target: "gettingOrganization",
                    actions: ["assignTemplate", "clearGetTemplateError"],
                  },
                  onError: {
                    target: "error",
                    actions: "assignGetTemplateError",
                  },
                },
                tags: "loading",
              },
              gettingOrganization: {
                invoke: {
                  src: "getOrganization",
                  id: "getOrganization",
                  onDone: {
                    target: "ready",
                    actions: ["assignOrganization", "clearGetOrganizationError"],
                  },
                  onError: {
                    target: "error",
                    actions: "assignGetOrganizationError",
                  },
                },
                tags: "loading",
              },
              error: {},
              ready: {},
            },
          },
          build: {
            initial: "idle",
            states: {
              idle: {
                on: {
                  START: "requestingStart",
                  STOP: "requestingStop",
                  RETRY: [{ cond: "triedToStart", target: "requestingStart" }, { target: "requestingStop" }],
                  UPDATE: "refreshingTemplate",
                },
              },
              requestingStart: {
                entry: "clearBuildError",
                invoke: {
                  id: "startWorkspace",
                  src: "startWorkspace",
                  onDone: {
                    target: "idle",
                    actions: ["assignBuild", "refreshTimeline"],
                  },
                  onError: {
                    target: "idle",
                    actions: ["assignBuildError", "displayBuildError"],
                  },
                },
              },
              requestingStop: {
                entry: "clearBuildError",
                invoke: {
                  id: "stopWorkspace",
                  src: "stopWorkspace",
                  onDone: {
                    target: "idle",
                    actions: ["assignBuild", "refreshTimeline"],
                  },
                  onError: {
                    target: "idle",
                    actions: ["assignBuildError", "displayBuildError"],
                  },
                },
              },
              refreshingTemplate: {
                entry: "clearRefreshTemplateError",
                invoke: {
                  id: "refreshTemplate",
                  src: "getTemplate",
                  onDone: {
                    target: "requestingStart",
                    actions: "assignTemplate",
                  },
                  onError: {
                    target: "idle",
                    actions: ["assignRefreshTemplateError", "displayRefreshTemplateError"],
                  },
                },
              },
            },
          },

          timeline: {
            initial: "gettingBuilds",
            states: {
              idle: {},
              gettingBuilds: {
                entry: "clearGetBuildsError",
                invoke: {
                  src: "getBuilds",
                  onDone: {
                    actions: ["assignBuilds"],
                    target: "loadedBuilds",
                  },
                  onError: {
                    actions: ["assignGetBuildsError"],
                    target: "idle",
                  },
                },
              },
              loadedBuilds: {
                initial: "idle",
                states: {
                  idle: {
                    on: {
                      LOAD_MORE_BUILDS: {
                        target: "loadingMoreBuilds",
                        cond: "hasMoreBuilds",
                      },
                      REFRESH_TIMELINE: "#workspaceState.ready.timeline.gettingBuilds",
                    },
                  },
                  loadingMoreBuilds: {
                    entry: "clearLoadMoreBuildsError",
                    invoke: {
                      src: "loadMoreBuilds",
                      onDone: {
                        actions: ["assignNewBuilds"],
                        target: "idle",
                      },
                      onError: {
                        actions: ["assignLoadMoreBuildsError"],
                        target: "idle",
                      },
                    },
                  },
                },
              },
            },
          },
        },
      },
      error: {
        on: {
          GET_WORKSPACE: "gettingWorkspace",
        },
      },
    },
  },
  {
    actions: {
      // Clear data about an old workspace when looking at a new one
      clearContext: () =>
        assign({
          workspace: undefined,
          template: undefined,
          organization: undefined,
          build: undefined,
        }),
      assignWorkspace: assign({
        workspace: (_, event) => event.data,
      }),
      assignGetWorkspaceError: assign({
        getWorkspaceError: (_, event) => event.data,
      }),
      clearGetWorkspaceError: (context) => assign({ ...context, getWorkspaceError: undefined }),
      assignTemplate: assign({
        template: (_, event) => event.data,
      }),
      assignGetTemplateError: assign({
        getTemplateError: (_, event) => event.data,
      }),
      clearGetTemplateError: (context) => assign({ ...context, getTemplateError: undefined }),
      assignOrganization: assign({
        organization: (_, event) => event.data,
      }),
      assignGetOrganizationError: assign({
        getOrganizationError: (_, event) => event.data,
      }),
      clearGetOrganizationError: (context) => assign({ ...context, getOrganizationError: undefined }),
      assignBuild: (_, event) =>
        assign({
          build: event.data,
        }),
      assignBuildError: (_, event) =>
        assign({
          buildError: event.data,
        }),
      displayBuildError: () => {
        displayError(Language.buildError)
      },
      clearBuildError: (_) =>
        assign({
          buildError: undefined,
        }),
      assignRefreshWorkspaceError: (_, event) =>
        assign({
          refreshWorkspaceError: event.data,
        }),
      clearRefreshWorkspaceError: (_) =>
        assign({
          refreshWorkspaceError: undefined,
        }),
      assignRefreshTemplateError: (_, event) =>
        assign({
          refreshTemplateError: event.data,
        }),
      displayRefreshTemplateError: () => {
        displayError(Language.refreshTemplateError)
      },
      clearRefreshTemplateError: (_) =>
        assign({
          refreshTemplateError: undefined,
        }),
      // Timeline
      assignBuilds: assign({
        builds: (_, event) => event.data,
      }),
      assignGetBuildsError: assign({
        getBuildsError: (_, event) => event.data,
      }),
      clearGetBuildsError: assign({
        getBuildsError: (_) => undefined,
      }),
      assignNewBuilds: assign({
        builds: (context, event) => {
          const oldBuilds = context.builds

          if (!oldBuilds) {
            throw new Error("Builds not loaded")
          }

          return [...oldBuilds, ...event.data]
        },
      }),
      assignLoadMoreBuildsError: assign({
        loadMoreBuildsError: (_, event) => event.data,
      }),
      clearLoadMoreBuildsError: assign({
        loadMoreBuildsError: (_) => undefined,
      }),
      refreshTimeline: pure((context, event) => {
        // No need to refresh the timeline if it is not loaded
        if (!context.builds) {
          return
        }
        // When it is a refresh workspace event, we want to check if the latest
        // build was updated to not over fetch the builds
        if (event.type === "done.invoke.refreshWorkspace") {
          const latestBuildInTimeline = latestBuild(context.builds)
          const isUpdated = event.data?.latest_build.updated_at !== latestBuildInTimeline.updated_at
          if (isUpdated) {
            return send({ type: "REFRESH_TIMELINE" })
          }
        } else {
          return send({ type: "REFRESH_TIMELINE" })
        }
      }),
    },
    guards: {
      triedToStart: (context) => context.workspace?.latest_build.transition === "start",
      hasMoreBuilds: (_) => false,
    },
    services: {
      getWorkspace: async (_, event) => {
        return await API.getWorkspace(event.workspaceId)
      },
      getTemplate: async (context) => {
        if (context.workspace) {
          return await API.getTemplate(context.workspace.template_id)
        } else {
          throw Error("Cannot get template without workspace")
        }
      },
      getOrganization: async (context) => {
        if (context.template) {
          return await API.getOrganization(context.template.organization_id)
        } else {
          throw Error("Cannot get organization without template")
        }
      },
      startWorkspace: async (context) => {
        if (context.workspace) {
          return await API.startWorkspace(context.workspace.id, context.template?.active_version_id)
        } else {
          throw Error("Cannot start workspace without workspace id")
        }
      },
      stopWorkspace: async (context) => {
        if (context.workspace) {
          return await API.stopWorkspace(context.workspace.id)
        } else {
          throw Error("Cannot stop workspace without workspace id")
        }
      },
      refreshWorkspace: async (context) => {
        if (context.workspace) {
          return await API.getWorkspace(context.workspace.id)
        } else {
          throw Error("Cannot refresh workspace without id")
        }
      },
      getBuilds: async (context) => {
        if (context.workspace) {
          return await API.getWorkspaceBuilds(context.workspace.id)
        } else {
          throw Error("Cannot refresh workspace without id")
        }
      },
      loadMoreBuilds: async (context) => {
        if (context.workspace) {
          return await API.getWorkspaceBuilds(context.workspace.id)
        } else {
          throw Error("Cannot refresh workspace without id")
        }
      },
    },
  },
)
