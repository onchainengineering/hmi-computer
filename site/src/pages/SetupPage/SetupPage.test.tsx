import { fireEvent, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { rest } from "msw"
import {
  renderWithAuth,
  waitForLoaderToBeRemoved,
} from "testHelpers/renderHelpers"
import { server } from "testHelpers/server"
import { SetupPage } from "./SetupPage"
import { Language as PageViewLanguage } from "./SetupPageView"
import { MockUser } from "testHelpers/entities"

const fillForm = async ({
  username = "someuser",
  email = "someone@coder.com",
  password = "password",
}: {
  username?: string
  email?: string
  password?: string
} = {}) => {
  const usernameField = await screen.findByLabelText(
    PageViewLanguage.usernameLabel,
  )
  const emailField = await screen.findByLabelText(PageViewLanguage.emailLabel)
  const passwordField = await screen.findByLabelText(
    PageViewLanguage.passwordLabel,
  )
  await userEvent.type(usernameField, username)
  await userEvent.type(emailField, email)
  await userEvent.type(passwordField, password)
  const submitButton = screen.getByRole("button", {
    name: PageViewLanguage.create,
  })
  fireEvent.click(submitButton)
}

describe("Setup Page", () => {
  beforeEach(() => {
    // appear logged out
    server.use(
      rest.get("/api/v2/users/me", (req, res, ctx) => {
        return res(ctx.status(401), ctx.json({ message: "no user here" }))
      }),
      rest.get("/api/v2/users/first", (req, res, ctx) => {
        return res(
          ctx.status(404),
          ctx.json({ message: "no first user has been created" }),
        )
      }),
    )
  })

  it("shows validation error message", async () => {
    renderWithAuth(<SetupPage />, { route: "/setup", path: "/setup" })
    await waitForLoaderToBeRemoved()
    await fillForm({ email: "test" })
    const errorMessage = await screen.findByText(PageViewLanguage.emailInvalid)
    expect(errorMessage).toBeDefined()
  })

  it("shows API error message", async () => {
    const fieldErrorMessage = "invalid username"
    server.use(
      rest.post("/api/v2/users/first", async (req, res, ctx) => {
        return res(
          ctx.status(400),
          ctx.json({
            message: "invalid field",
            validations: [
              {
                detail: fieldErrorMessage,
                field: "username",
              },
            ],
          }),
        )
      }),
    )

    renderWithAuth(<SetupPage />, { route: "/setup", path: "/setup" })
    await waitForLoaderToBeRemoved()
    await fillForm()
    const errorMessage = await screen.findByText(fieldErrorMessage)
    expect(errorMessage).toBeDefined()
  })

  it("redirects to login if setup has already completed", async () => {
    // simulates setup having already been completed
    server.use(
      rest.get("/api/v2/users/first", (req, res, ctx) => {
        return res(ctx.status(404), ctx.json({ message: "hooray, you exist!" }))
      }),
    )

    renderWithAuth(<SetupPage />, { route: "/setup", path: "/setup" })
    await waitForLoaderToBeRemoved()
    await waitFor(() => expect(window.location).toBeAt("/login"))
  })

  it("redirects to workspaces page when success", async () => {
    // simulates the user will be authenticated
    server.use(
      rest.get("/api/v2/users/me", (req, res, ctx) => {
        return res(ctx.status(200), ctx.json(MockUser))
      }),
      rest.get("/api/v2/users/first", (req, res, ctx) => {
        return res(ctx.status(404), ctx.json({ message: "hooray, you exist!" }))
      }),
    )

    renderWithAuth(<SetupPage />, { route: "/setup", path: "/setup" })
    await waitForLoaderToBeRemoved()

    await fillForm()
    await waitFor(() => expect(window.location).toBeAt("/workspaces"))
  })
})
