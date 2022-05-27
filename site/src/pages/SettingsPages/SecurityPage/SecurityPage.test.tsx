import { fireEvent, screen, waitFor } from "@testing-library/react"
import React from "react"
import * as API from "../../../api/api"
import { GlobalSnackbar } from "../../../components/GlobalSnackbar/GlobalSnackbar"
import * as SecurityForm from "../../../components/SettingsSecurityForm/SettingsSecurityForm"
import { renderWithAuth } from "../../../testHelpers/renderHelpers"
import * as AuthXService from "../../../xServices/auth/authXService"
import { Language, SecurityPage } from "./SecurityPage"

const renderPage = () => {
  return renderWithAuth(
    <>
      <SecurityPage />
      <GlobalSnackbar />
    </>,
  )
}

const newData = {
  old_password: "password1",
  password: "password2",
  confirm_password: "password2",
}

const fillAndSubmitForm = async () => {
  await waitFor(() => screen.findByLabelText("Old Password"))
  fireEvent.change(screen.getByLabelText("Old Password"), { target: { value: newData.old_password } })
  fireEvent.change(screen.getByLabelText("New Password"), { target: { value: newData.password } })
  fireEvent.change(screen.getByLabelText("Confirm Password"), { target: { value: newData.confirm_password } })
  fireEvent.click(screen.getByText(SecurityForm.Language.updatePassword))
}

describe("SecurityPage", () => {
  describe("when it is a success", () => {
    it("shows the success message", async () => {
      jest.spyOn(API, "updateUserPassword").mockImplementationOnce((_userId, _data) => Promise.resolve(undefined))
      const { user } = renderPage()
      await fillAndSubmitForm()

      const successMessage = await screen.findByText(AuthXService.Language.successSecurityUpdate)
      expect(successMessage).toBeDefined()
      expect(API.updateUserPassword).toBeCalledTimes(1)
      expect(API.updateUserPassword).toBeCalledWith(user.id, newData)
    })
  })

  describe("when the old_password is incorrect", () => {
    it("shows an error", async () => {
      jest.spyOn(API, "updateUserPassword").mockRejectedValueOnce({
        isAxiosError: true,
        response: {
          data: { message: "Incorrect password.", errors: [{ detail: "Incorrect password.", field: "old_password" }] },
        },
      })

      const { user } = renderPage()
      await fillAndSubmitForm()

      const errorMessage = await screen.findByText("Incorrect password.")
      expect(errorMessage).toBeDefined()
      expect(API.updateUserPassword).toBeCalledTimes(1)
      expect(API.updateUserPassword).toBeCalledWith(user.id, newData)
    })
  })

  describe("when the password is invalid", () => {
    it("shows an error", async () => {
      jest.spyOn(API, "updateUserPassword").mockRejectedValueOnce({
        isAxiosError: true,
        response: {
          data: { message: "Invalid password.", errors: [{ detail: "Invalid password.", field: "password" }] },
        },
      })

      const { user } = renderPage()
      await fillAndSubmitForm()

      const errorMessage = await screen.findByText("Invalid password.")
      expect(errorMessage).toBeDefined()
      expect(API.updateUserPassword).toBeCalledTimes(1)
      expect(API.updateUserPassword).toBeCalledWith(user.id, newData)
    })
  })

  describe("when it is an unknown error", () => {
    it("shows a generic error message", async () => {
      jest.spyOn(API, "updateUserPassword").mockRejectedValueOnce({
        data: "unknown error",
      })

      const { user } = renderPage()
      await fillAndSubmitForm()

      const errorMessage = await screen.findByText(Language.unknownError)
      expect(errorMessage).toBeDefined()
      expect(API.updateUserPassword).toBeCalledTimes(1)
      expect(API.updateUserPassword).toBeCalledWith(user.id, newData)
    })
  })
})
