import { useRouter } from "next/router"
import React, { useContext, useEffect } from "react"
import useSWR from "swr"

import * as API from "../api"

export interface User {
  readonly id: string
  readonly username: string
  readonly email: string
  readonly created_at: string
}

export interface UserContext {
  readonly error?: Error
  readonly me?: User
  readonly signOut: () => Promise<void>
}

const UserContext = React.createContext<UserContext>({
  signOut: () => {
    return Promise.reject("Sign out API not available")
  },
})

export const useUser = (redirectOnError = false): UserContext => {
  const ctx = useContext(UserContext)
  const { push, asPath } = useRouter()

  const requestError = ctx.error
  useEffect(() => {
    if (redirectOnError && requestError) {
      // 'void' means we are ignoring handling the promise returned
      // from router.push (and lets the linter know we're OK with that!)
      void push({
        pathname: "/login",
        query: {
          redirect: asPath,
        },
      })
    }
    // Disabling exhaustive deps here because it can cause an
    // infinite useEffect loop. Should (hopefully) go away
    // when we switch to an alternate routing strategy.
  }, [redirectOnError, requestError]) // eslint-disable-line react-hooks/exhaustive-deps

  return ctx
}

export const UserProvider: React.FC = (props) => {
  const router = useRouter()
  const { data, error, mutate } = useSWR("/api/v2/users/me")

  const signOut = async () => {
    await API.logout()
    // Tell SWR to invalidate the cache for the user endpoint
    await mutate("/api/v2/users/me")
    await router.push({
      pathname: "/login",
      query: {
        redirect: router.asPath,
      },
    })
  }

  return (
    <UserContext.Provider
      value={{
        error: error,
        me: data,
        signOut: signOut,
      }}
    >
      {props.children}
    </UserContext.Provider>
  )
}
