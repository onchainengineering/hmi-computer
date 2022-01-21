import singletonRouter from "next/router"
import mockRouter from "next-router-mock"
import React from "react"
import { SWRConfig } from "swr"
import { render, screen, waitFor } from "@testing-library/react"

import { User, UserProvider, useUser } from "./UserContext"

namespace Helpers {
  const TestComponent: React.FC<{ redirectOnFailure: boolean }> = ({ redirectOnFailure }) => {
    const { me, error } = useUser(redirectOnFailure)

    if (!!error) {
      return <div>{`Error: ${error.toString()}`}</div>
    }
    if (!!me) {
      return <div>{`Me: ${me.toString()}`}</div>
    }

    return <div>Loading</div>
  }

  export const renderUserContext = (simulatedRequest: () => Promise<User>, redirectOnFailure: boolean) => {
    return (
      <SWRConfig
        value={{
          fetcher: simulatedRequest,
        }}
      >
        <UserProvider>
          <TestComponent redirectOnFailure={redirectOnFailure} />
        </UserProvider>
      </SWRConfig>
    )
  }

  export const mockUser: User = {
    id: "test-user-id",
    username: "TestUser",
    email: "test@coder.com",
    created_at: "",
  }
}

describe("UserContext", () => {
  const failingRequest = () => Promise.reject("Failed to load user")
  const successfulRequest = () => Promise.resolve(Helpers.mockUser)

  // Reset the router to '/' before every test
  beforeEach(() => {
    mockRouter.setCurrentUrl("/")
  })

  /*it("shouldn't redirect if user fails to load and redirectOnFailure is false", async () => {
    // When
    render(Helpers.renderUserContext(failingRequest, false))

    // Then
    // Verify we get an error message
    await waitFor(() => {
      expect(screen.queryByText("Error:", { exact: false })).toBeDefined()
    })
    // ...and the route should be unchanged
    expect(singletonRouter).toMatchObject({ asPath: "/" })
  })

  it("should redirect if user fails to load and redirectOnFailure is true", async () => {
    // When
    render(Helpers.renderUserContext(failingRequest, true))

    // Then
    // Verify we route to the login page
    await waitFor(() => expect(singletonRouter).toMatchObject({ asPath: "/login?redirect=%2F" }))
  })*/

  it("should not redirect if user loads and redirectOnFailure is true", async () => {
    // When
    render(Helpers.renderUserContext(successfulRequest, true))

    // Then
    // Verify the user is rendered
    await waitFor(() => {
      expect(screen.queryByText("Me:", { exact: false })).toBeDefined()
    })
    // ...and the route should be unchanged
    expect(singletonRouter).toMatchObject({ asPath: "/" })
  })
})
