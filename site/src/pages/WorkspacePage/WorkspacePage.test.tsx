import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import EventSourceMock from "eventsourcemock";
import { rest } from "msw";
import {
  MockTemplate,
  MockWorkspace,
  MockWorkspaceBuild,
  MockStoppedWorkspace,
  MockStartingWorkspace,
  MockOutdatedWorkspace,
  MockTemplateVersionParameter1,
  MockTemplateVersionParameter2,
  MockBuilds,
  MockUser,
  MockDeploymentConfig,
  MockWorkspaceBuildDelete,
} from "testHelpers/entities";
import * as api from "api/api";
import { renderWithAuth } from "testHelpers/renderHelpers";
import { server } from "testHelpers/server";
import { WorkspacePage } from "./WorkspacePage";

// It renders the workspace page and waits for it be loaded
const renderWorkspacePage = async () => {
  jest.spyOn(api, "getTemplate").mockResolvedValueOnce(MockTemplate);
  jest.spyOn(api, "getTemplateVersionRichParameters").mockResolvedValueOnce([]);
  jest
    .spyOn(api, "getDeploymentConfig")
    .mockResolvedValueOnce(MockDeploymentConfig);
  jest
    .spyOn(api, "watchWorkspaceAgentLogs")
    .mockImplementation((_, options) => {
      options.onDone && options.onDone();
      return new WebSocket("");
    });
  renderWithAuth(<WorkspacePage />, {
    route: `/@${MockWorkspace.owner_name}/${MockWorkspace.name}`,
    path: "/:username/:workspace",
  });
  await screen.findByText(MockWorkspace.name);
};

/**
 * Requests and responses related to workspace status are unrelated, so we can't test in the usual way.
 * Instead, test that button clicks produce the correct requests and that responses produce the correct UI.
 * We don't need to test the UI exhaustively because Storybook does that; just enough to prove that the
 * workspaceStatus was calculated correctly.
 */
const testButton = async (label: string, actionMock: jest.SpyInstance) => {
  const user = userEvent.setup();
  await renderWorkspacePage();
  const workspaceActions = screen.getByTestId("workspace-actions");
  const button = within(workspaceActions).getByRole("button", { name: label });
  await user.click(button);
  expect(actionMock).toBeCalled();
};

let originalEventSource: typeof window.EventSource;

beforeAll(() => {
  originalEventSource = window.EventSource;
  // mocking out EventSource for SSE
  window.EventSource = EventSourceMock;
});

beforeEach(() => {
  jest.resetAllMocks();
});

afterAll(() => {
  window.EventSource = originalEventSource;
});

describe("WorkspacePage", () => {
  it("requests a delete job when the user presses Delete and confirms", async () => {
    const user = userEvent.setup({ delay: 0 });
    const deleteWorkspaceMock = jest
      .spyOn(api, "deleteWorkspace")
      .mockResolvedValueOnce(MockWorkspaceBuild);
    await renderWorkspacePage();

    // open the workspace action popover so we have access to all available ctas
    const trigger = screen.getByTestId("workspace-options-button");
    await user.click(trigger);

    // Click on delete
    const button = await screen.findByTestId("delete-button");
    await user.click(button);

    // Get dialog and confirm
    const dialog = await screen.findByTestId("dialog");
    const labelText = "Workspace name";
    const textField = within(dialog).getByLabelText(labelText);
    await user.type(textField, MockWorkspace.name);
    const confirmButton = within(dialog).getByRole("button", {
      name: "Delete",
      hidden: false,
    });
    await user.click(confirmButton);
    expect(deleteWorkspaceMock).toBeCalled();
  });

  it("orphans the workspace on delete if option is selected", async () => {
    const user = userEvent.setup({ delay: 0 });

    // set permissions
    server.use(
      rest.post("/api/v2/authcheck", async (req, res, ctx) => {
        return res(
          ctx.status(200),
          ctx.json({
            updateTemplates: true,
            updateWorkspace: true,
            updateTemplate: true,
          }),
        );
      }),
    );

    const deleteWorkspaceMock = jest
      .spyOn(api, "deleteWorkspace")
      .mockResolvedValueOnce(MockWorkspaceBuildDelete);
    await renderWorkspacePage();

    // open the workspace action popover so we have access to all available ctas
    const trigger = screen.getByTestId("workspace-options-button");
    await user.click(trigger);

    // Click on delete
    const button = await screen.findByTestId("delete-button");
    await user.click(button);

    // Get dialog and enter confirmation text
    const dialog = await screen.findByTestId("dialog");
    const labelText = "Workspace name";
    const textField = within(dialog).getByLabelText(labelText);
    await user.type(textField, MockWorkspace.name);

    // check orphan option
    const orphanCheckbox = within(
      screen.getByTestId("orphan-checkbox"),
    ).getByRole("checkbox");

    await user.click(orphanCheckbox);

    // confirm
    const confirmButton = within(dialog).getByRole("button", {
      name: "Delete",
      hidden: false,
    });
    await user.click(confirmButton);
    // arguments are workspace.name, log level (undefined), and orphan
    expect(deleteWorkspaceMock).toBeCalledWith(MockWorkspace.id, {
      log_level: undefined,
      orphan: true,
    });
  });

  it("requests a start job when the user presses Start", async () => {
    server.use(
      rest.get(
        `/api/v2/users/:userId/workspace/:workspaceName`,
        (req, res, ctx) => {
          return res(ctx.status(200), ctx.json(MockStoppedWorkspace));
        },
      ),
    );
    const startWorkspaceMock = jest
      .spyOn(api, "startWorkspace")
      .mockImplementation(() => Promise.resolve(MockWorkspaceBuild));
    await testButton("Start", startWorkspaceMock);
  });

  it("requests a stop job when the user presses Stop", async () => {
    const stopWorkspaceMock = jest
      .spyOn(api, "stopWorkspace")
      .mockResolvedValueOnce(MockWorkspaceBuild);

    await testButton("Stop", stopWorkspaceMock);
  });

  it("requests a stop when the user presses Restart", async () => {
    const stopWorkspaceMock = jest
      .spyOn(api, "stopWorkspace")
      .mockResolvedValueOnce(MockWorkspaceBuild);

    // Render
    await renderWorkspacePage();

    // Actions
    const user = userEvent.setup();
    await user.click(screen.getByTestId("workspace-restart-button"));
    const confirmButton = await screen.findByTestId("confirm-button");
    await user.click(confirmButton);

    // Assertions
    await waitFor(() => {
      expect(stopWorkspaceMock).toBeCalled();
    });
  });

  it("requests cancellation when the user presses Cancel", async () => {
    server.use(
      rest.get(
        `/api/v2/users/:userId/workspace/:workspaceName`,
        (req, res, ctx) => {
          return res(ctx.status(200), ctx.json(MockStartingWorkspace));
        },
      ),
    );
    const cancelWorkspaceMock = jest
      .spyOn(api, "cancelWorkspaceBuild")
      .mockImplementation(() => Promise.resolve({ message: "job canceled" }));

    await renderWorkspacePage();

    const workspaceActions = screen.getByTestId("workspace-actions");
    const cancelButton = within(workspaceActions).getByRole("button", {
      name: "Cancel",
    });

    await userEvent.click(cancelButton);

    expect(cancelWorkspaceMock).toBeCalled();
  });

  it("requests an update when the user presses Update", async () => {
    // Mocks
    jest
      .spyOn(api, "getWorkspaceByOwnerAndName")
      .mockResolvedValueOnce(MockOutdatedWorkspace);

    const updateWorkspaceMock = jest
      .spyOn(api, "updateWorkspace")
      .mockResolvedValueOnce(MockWorkspaceBuild);

    // Render
    await renderWorkspacePage();

    // Actions
    const user = userEvent.setup();
    await user.click(screen.getByTestId("workspace-update-button"));
    const confirmButton = await screen.findByTestId("confirm-button");
    await user.click(confirmButton);

    // Assertions
    await waitFor(() => {
      expect(updateWorkspaceMock).toBeCalled();
    });
  });

  it("updates the parameters when they are missing during update", async () => {
    // Mocks
    jest
      .spyOn(api, "getWorkspaceByOwnerAndName")
      .mockResolvedValueOnce(MockOutdatedWorkspace);
    const updateWorkspaceSpy = jest
      .spyOn(api, "updateWorkspace")
      .mockRejectedValueOnce(
        new api.MissingBuildParameters(
          [MockTemplateVersionParameter1, MockTemplateVersionParameter2],
          MockOutdatedWorkspace.template_active_version_id,
        ),
      );

    // Render
    await renderWorkspacePage();

    // Actions
    const user = userEvent.setup();
    await user.click(screen.getByTestId("workspace-update-button"));
    const confirmButton = await screen.findByTestId("confirm-button");
    await user.click(confirmButton);

    // The update was called
    await waitFor(() => {
      expect(api.updateWorkspace).toBeCalled();
      updateWorkspaceSpy.mockClear();
    });

    // After trying to update, a new dialog asking for missed parameters should
    // be displayed and filled
    const dialog = await screen.findByTestId("dialog");
    const firstParameterInput = within(dialog).getByLabelText(
      MockTemplateVersionParameter1.name,
      { exact: false },
    );
    await user.clear(firstParameterInput);
    await user.type(firstParameterInput, "some-value");
    const secondParameterInput = within(dialog).getByLabelText(
      MockTemplateVersionParameter2.name,
      { exact: false },
    );
    await user.clear(secondParameterInput);
    await user.type(secondParameterInput, "2");
    await user.click(within(dialog).getByRole("button", { name: "Update" }));

    // Check if the update was called using the values from the form
    await waitFor(() => {
      expect(api.updateWorkspace).toBeCalledWith(MockOutdatedWorkspace, [
        {
          name: MockTemplateVersionParameter1.name,
          value: "some-value",
        },
        {
          name: MockTemplateVersionParameter2.name,
          value: "2",
        },
      ]);
    });
  });

  it("shows the timeline build", async () => {
    await renderWorkspacePage();
    const table = await screen.findByTestId("builds-table");

    // Wait for the results to be loaded
    await waitFor(async () => {
      const rows = table.querySelectorAll("tbody > tr");
      // Added +1 because of the date row
      expect(rows).toHaveLength(MockBuilds.length + 1);
    });
  });

  it("restart the workspace with one time parameters when having the confirmation dialog", async () => {
    window.localStorage.removeItem(`${MockUser.id}_ignoredWarnings`);
    jest.spyOn(api, "getWorkspaceParameters").mockResolvedValue({
      templateVersionRichParameters: [
        {
          ...MockTemplateVersionParameter1,
          ephemeral: true,
          name: "rebuild",
          description: "Rebuild",
          required: false,
        },
      ],
      buildParameters: [{ name: "rebuild", value: "false" }],
    });
    const restartWorkspaceSpy = jest.spyOn(api, "restartWorkspace");
    const user = userEvent.setup();
    await renderWorkspacePage();
    await user.click(screen.getByTestId("build-parameters-button"));
    const buildParametersForm = await screen.findByTestId(
      "build-parameters-form",
    );
    const rebuildField = within(buildParametersForm).getByLabelText("Rebuild", {
      exact: false,
    });
    await user.clear(rebuildField);
    await user.type(rebuildField, "true");
    await user.click(screen.getByTestId("build-parameters-submit"));
    await user.click(screen.getByTestId("confirm-button"));
    await waitFor(() => {
      expect(restartWorkspaceSpy).toBeCalledWith({
        workspace: MockWorkspace,
        buildParameters: [{ name: "rebuild", value: "true" }],
      });
    });
  });
});
