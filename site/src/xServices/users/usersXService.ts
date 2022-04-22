import { assign, createMachine } from "xstate"
import * as API from "../../api"
import * as Types from "../../api/types"
import * as GenTypes from "../../api/typesGenerated"
import { displayError } from "../../components/GlobalSnackbar/utils"

const Language = {
  createUserError: "Unable to create user",
  createUserSuccess: "Successfully created user"
}

export interface UsersContext {
  users: Types.UserResponse[]
  pager?: Types.Pager
  getUsersError?: Error | unknown
  createUserError?: Error | unknown
}

export type UsersEvent = { type: "GET_USERS" } | { type: "CREATE", user: GenTypes.CreateUserRequest }

export const usersMachine = createMachine(
  {
    tsTypes: {} as import("./usersXService.typegen").Typegen0,
    schema: {
      context: {} as UsersContext,
      events: {} as UsersEvent,
      services: {} as {
        getUsers: {
          data: Types.PagedUsers
        },
        createUser: {
          data: GenTypes.User
        }
      },
    },
    id: "usersState",
    context: {
      users: [],
    },
    initial: "idle",
    states: {
      idle: {
        on: {
          GET_USERS: "gettingUsers",
          CREATE: "creatingUser"
        },
      },
      gettingUsers: {
        invoke: {
          src: "getUsers",
          id: "getUsers",
          onDone: [
            {
              target: "#usersState.idle",
              actions: ["assignUsers", "clearGetUsersError"],
            },
          ],
          onError: [
            {
              actions: "assignGetUsersError",
              target: "#usersState.error",
            },
          ],
        },
        tags: "loading",
      },
      creatingUser: {
        invoke: {
          src: "createUser",
          id: "createUser",
          onDone: {
            target: "gettingUsers",
            actions: "displayCreateUserSuccess"
          },
          onError: {
            target: "idle",
            actions: "displayCreateUserError"
          }
        },
        tags: "loading"
      },
      error: {
        on: {
          GET_USERS: "gettingUsers",
        },
      },
    },
  },
  {
    services: {
      getUsers: API.getUsers,
      createUser: (_, event) => (
        API.createUser(event.user)
      )
    },
    actions: {
      assignUsers: assign({
        users: (_, event) => event.data.page,
        pager: (_, event) => event.data.pager,
      }),
      assignGetUsersError: assign({
        getUsersError: (_, event) => event.data,
      }),
      clearGetUsersError: assign((context: UsersContext) => ({
        ...context,
        getUsersError: undefined,
      })),
      displayCreateUserError: () => {
        displayError(Language.createUserError)
      },
      displayCreateUserSuccess: () => {
        displayError(Language.createUserSuccess)
      }
    },
  },
)
