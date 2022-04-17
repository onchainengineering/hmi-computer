import { useInterpret } from "@xstate/react"
import React, { createContext } from "react"
import { ActorRefFrom } from "xstate"
import { authMachine } from "./auth/authXService"
import { buildInfoMachine } from "./buildInfo/buildInfoXService"
import { usersMachine } from "./users/usersXService"

interface XServiceContextType {
  authXService: ActorRefFrom<typeof authMachine>
  buildInfoXService: ActorRefFrom<typeof buildInfoMachine>
  usersXService: ActorRefFrom<typeof usersMachine>
}

/**
 * Consuming this Context will not automatically cause rerenders because
 * the xServices in it are static references.
 *
 * To use one of the xServices, `useActor` will access all its state
 * (causing re-renders for any changes to that one xService) and
 * `useSelector` will access just one piece of state.
 */
export const XServiceContext = createContext({} as XServiceContextType)

export const XServiceProvider: React.FC = ({ children }) => {
  return (
    <XServiceContext.Provider
      value={{
        authXService: useInterpret(authMachine),
        buildInfoXService: useInterpret(buildInfoMachine),
        usersXService: useInterpret(usersMachine),
      }}
    >
      {children}
    </XServiceContext.Provider>
  )
}
