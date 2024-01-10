import { type FC, type ReactNode, Fragment } from "react";
import { Workspace, WorkspaceBuildParameter } from "api/typesGenerated";
import { useWorkspaceDuplication } from "pages/CreateWorkspacePage/useWorkspaceDuplication";
import { workspaceUpdatePolicy } from "utils/workspace";
import { type ActionType, abilitiesByWorkspaceStatus } from "./constants";
import {
  CancelButton,
  DisabledButton,
  StartButton,
  StopButton,
  RestartButton,
  UpdateButton,
  ActivateButton,
  RetryButton,
} from "./Buttons";

import Divider from "@mui/material/Divider";
import DuplicateIcon from "@mui/icons-material/FileCopyOutlined";
import SettingsIcon from "@mui/icons-material/SettingsOutlined";
import HistoryIcon from "@mui/icons-material/HistoryOutlined";
import DeleteIcon from "@mui/icons-material/DeleteOutlined";
import {
  MoreMenu,
  MoreMenuContent,
  MoreMenuItem,
  MoreMenuTrigger,
} from "components/MoreMenu/MoreMenu";
import { TopbarIconButton } from "components/FullPageLayout/Topbar";
import MoreVertOutlined from "@mui/icons-material/MoreVertOutlined";
import { useMe } from "hooks";

export interface WorkspaceActionsProps {
  workspace: Workspace;
  handleStart: (buildParameters?: WorkspaceBuildParameter[]) => void;
  handleStop: () => void;
  handleRestart: (buildParameters?: WorkspaceBuildParameter[]) => void;
  handleDelete: () => void;
  handleUpdate: () => void;
  handleCancel: () => void;
  handleSettings: () => void;
  handleChangeVersion: () => void;
  handleRetry: () => void;
  handleRetryDebug: () => void;
  handleDormantActivate: () => void;
  isUpdating: boolean;
  isRestarting: boolean;
  children?: ReactNode;
  canChangeVersions: boolean;
  canRetryDebug: boolean;
}

export const WorkspaceActions: FC<WorkspaceActionsProps> = ({
  workspace,
  handleStart,
  handleStop,
  handleRestart,
  handleDelete,
  handleUpdate,
  handleCancel,
  handleSettings,
  handleRetry,
  handleRetryDebug,
  handleChangeVersion,
  handleDormantActivate,
  isUpdating,
  isRestarting,
  canChangeVersions,
  canRetryDebug,
}) => {
  const me = useMe();
  const isOwner = me.roles.find((role) => role.name === "owner");

  const { duplicateWorkspace, isDuplicationReady } =
    useWorkspaceDuplication(workspace);

  const { actions, canCancel, canAcceptJobs } = abilitiesByWorkspaceStatus(
    workspace,
    canRetryDebug,
  );
  const cancelEnabled = canCancel || isOwner

  const mustUpdate =
    workspaceUpdatePolicy(workspace, canChangeVersions) === "always" &&
    workspace.outdated;

  const tooltipText = getTooltipText(workspace, mustUpdate);
  const canBeUpdated = workspace.outdated && canAcceptJobs;

  // A mapping of button type to the corresponding React component
  const buttonMapping: Record<ActionType, ReactNode> = {
    update: <UpdateButton handleAction={handleUpdate} />,
    updating: <UpdateButton loading handleAction={handleUpdate} />,
    start: (
      <StartButton
        workspace={workspace}
        handleAction={handleStart}
        disabled={mustUpdate}
        tooltipText={tooltipText}
      />
    ),
    starting: (
      <StartButton
        loading
        workspace={workspace}
        handleAction={handleStart}
        disabled={mustUpdate}
        tooltipText={tooltipText}
      />
    ),
    stop: <StopButton handleAction={handleStop} />,
    stopping: <StopButton loading handleAction={handleStop} />,
    restart: (
      <RestartButton
        workspace={workspace}
        handleAction={handleRestart}
        disabled={mustUpdate}
        tooltipText={tooltipText}
      />
    ),
    restarting: (
      <RestartButton
        loading
        workspace={workspace}
        handleAction={handleRestart}
        disabled={mustUpdate}
        tooltipText={tooltipText}
      />
    ),
    deleting: <DisabledButton label="Deleting" />,
    canceling: <DisabledButton label="Canceling..." />,
    deleted: <DisabledButton label="Deleted" />,
    pending: <DisabledButton label="Pending..." />,
    activate: <ActivateButton handleAction={handleDormantActivate} />,
    activating: <ActivateButton loading handleAction={handleDormantActivate} />,
    retry: <RetryButton handleAction={handleRetry} />,
    retryDebug: <RetryButton debug handleAction={handleRetryDebug} />,
  };

  return (
    <div
      css={{ display: "flex", alignItems: "center", gap: 8 }}
      data-testid="workspace-actions"
    >
      {canBeUpdated && (
        <>{isUpdating ? buttonMapping.updating : buttonMapping.update}</>
      )}

      {isRestarting
        ? buttonMapping.restarting
        : actions.map((action) => (
            <Fragment key={action}>{buttonMapping[action]}</Fragment>
          ))}

      {cancelEnabled && <CancelButton handleAction={handleCancel} />}

      <MoreMenu>
        <MoreMenuTrigger>
          <TopbarIconButton
            title="More options"
            data-testid="workspace-options-button"
            aria-controls="workspace-options"
            disabled={!canAcceptJobs}
          >
            <MoreVertOutlined />
          </TopbarIconButton>
        </MoreMenuTrigger>

        <MoreMenuContent id="workspace-options">
          <MoreMenuItem onClick={handleSettings}>
            <SettingsIcon />
            Settings
          </MoreMenuItem>

          {canChangeVersions && (
            <MoreMenuItem onClick={handleChangeVersion}>
              <HistoryIcon />
              Change version&hellip;
            </MoreMenuItem>
          )}

          <MoreMenuItem
            onClick={duplicateWorkspace}
            disabled={!isDuplicationReady}
          >
            <DuplicateIcon />
            Duplicate&hellip;
          </MoreMenuItem>

          <Divider />

          <MoreMenuItem
            danger
            onClick={handleDelete}
            data-testid="delete-button"
          >
            <DeleteIcon />
            Delete&hellip;
          </MoreMenuItem>
        </MoreMenuContent>
      </MoreMenu>
    </div>
  );
};

function getTooltipText(workspace: Workspace, disabled: boolean): string {
  if (!disabled) {
    return "";
  }

  if (workspace.template_require_active_version) {
    return "This template requires automatic updates";
  }

  if (workspace.automatic_updates === "always") {
    return "You have enabled automatic updates for this workspace";
  }

  return "";
}
