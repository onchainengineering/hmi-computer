import { fireEvent, screen } from "@testing-library/react"
import i18next from "i18next"
import * as Mocks from "../../testHelpers/entities"
import { render } from "../../testHelpers/renderHelpers"
import { WorkspaceActions, WorkspaceActionsProps } from "./WorkspaceActions"

const { t } = i18next

const renderComponent = async (props: Partial<WorkspaceActionsProps> = {}) => {
  render(
    <WorkspaceActions
      workspace={props.workspace ?? Mocks.MockWorkspace}
      handleStart={jest.fn()}
      handleStop={jest.fn()}
      handleDelete={jest.fn()}
      handleUpdate={jest.fn()}
      handleCancel={jest.fn()}
      isUpdating={false}
    />,
  )
}

const renderAndClick = async (props: Partial<WorkspaceActionsProps> = {}) => {
  render(
    <WorkspaceActions
      workspace={props.workspace ?? Mocks.MockWorkspace}
      handleStart={jest.fn()}
      handleStop={jest.fn()}
      handleDelete={jest.fn()}
      handleUpdate={jest.fn()}
      handleCancel={jest.fn()}
      isUpdating={false}
    />,
  )
  const trigger = await screen.findByTestId("workspace-actions-button")
  fireEvent.click(trigger)
}

describe("WorkspaceActions", () => {
  describe("when the workspace is starting", () => {
    it("primary is starting; cancel is available; no secondary", async () => {
      await renderComponent({ workspace: Mocks.MockStartingWorkspace })
      expect(screen.getByTestId("primary-cta")).toHaveTextContent(
        t("actionButton.starting", { ns: "workspacePage" }),
      )
      expect(
        screen.getByRole("button", {
          name: "cancel action",
        }),
      ).toBeInTheDocument()
      expect(screen.queryByTestId("secondary-ctas")).toBeNull()
    })
  })
  describe("when the workspace is started", () => {
    it("primary is stop; secondary is delete", async () => {
      await renderAndClick({ workspace: Mocks.MockWorkspace })
      expect(screen.getByTestId("primary-cta")).toHaveTextContent(
        t("actionButton.stop", { ns: "workspacePage" }),
      )
      expect(screen.getByTestId("secondary-ctas")).toHaveTextContent(
        t("actionButton.delete", { ns: "workspacePage" }),
      )
    })
  })
  describe("when the workspace is stopping", () => {
    it("primary is stopping; cancel is available; no secondary", async () => {
      await renderComponent({ workspace: Mocks.MockStoppingWorkspace })
      expect(screen.getByTestId("primary-cta")).toHaveTextContent(
        t("actionButton.stopping", { ns: "workspacePage" }),
      )
      expect(
        screen.getByRole("button", {
          name: "cancel action",
        }),
      ).toBeInTheDocument()
      expect(screen.queryByTestId("secondary-ctas")).toBeNull()
    })
  })
  describe("when the workspace is canceling", () => {
    it("primary is canceling; no secondary", async () => {
      await renderAndClick({ workspace: Mocks.MockCancelingWorkspace })
      expect(screen.getByTestId("primary-cta")).toHaveTextContent(
        t("disabledButton.canceling", { ns: "workspacePage" }),
      )
      expect(screen.queryByTestId("secondary-ctas")).toBeNull()
    })
  })
  describe("when the workspace is canceled", () => {
    it("primary is start; secondary are stop, delete", async () => {
      await renderAndClick({ workspace: Mocks.MockCanceledWorkspace })
      expect(screen.getByTestId("primary-cta")).toHaveTextContent(
        t("actionButton.start", { ns: "workspacePage" }),
      )
      expect(screen.getByTestId("secondary-ctas")).toHaveTextContent(
        t("actionButton.stop", { ns: "workspacePage" }),
      )
      expect(screen.getByTestId("secondary-ctas")).toHaveTextContent(
        t("actionButton.delete", { ns: "workspacePage" }),
      )
    })
  })
  describe("when the workspace is errored", () => {
    it("primary is start; secondary is delete", async () => {
      await renderAndClick({ workspace: Mocks.MockFailedWorkspace })
      expect(screen.getByTestId("primary-cta")).toHaveTextContent(
        t("actionButton.start", { ns: "workspacePage" }),
      )
      expect(screen.getByTestId("secondary-ctas")).toHaveTextContent(
        t("actionButton.delete", { ns: "workspacePage" }),
      )
    })
  })
  describe("when the workspace is deleting", () => {
    it("primary is deleting; cancel is available; no secondary", async () => {
      await renderComponent({ workspace: Mocks.MockDeletingWorkspace })
      expect(screen.getByTestId("primary-cta")).toHaveTextContent(
        t("actionButton.deleting", { ns: "workspacePage" }),
      )
      expect(
        screen.getByRole("button", {
          name: "cancel action",
        }),
      ).toBeInTheDocument()
      expect(screen.queryByTestId("secondary-ctas")).toBeNull()
    })
  })
  describe("when the workspace is deleted", () => {
    it("primary is deleted; no secondary", async () => {
      await renderAndClick({ workspace: Mocks.MockDeletedWorkspace })
      expect(screen.getByTestId("primary-cta")).toHaveTextContent(
        t("disabledButton.deleted", { ns: "workspacePage" }),
      )
      expect(screen.queryByTestId("secondary-ctas")).toBeNull()
    })
  })
  describe("when the workspace is outdated", () => {
    it("primary is update; secondary are start, delete", async () => {
      await renderAndClick({ workspace: Mocks.MockOutdatedWorkspace })
      expect(screen.getByTestId("primary-cta")).toHaveTextContent(
        t("actionButton.update", { ns: "workspacePage" }),
      )
      expect(screen.getByTestId("secondary-ctas")).toHaveTextContent(
        t("actionButton.start", { ns: "workspacePage" }),
      )
      expect(screen.getByTestId("secondary-ctas")).toHaveTextContent(
        t("actionButton.delete", { ns: "workspacePage" }),
      )
    })
  })
})
