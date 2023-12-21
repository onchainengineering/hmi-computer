import { Link as RouterLink } from "react-router-dom";
import type * as TypesGen from "api/typesGenerated";
import { WorkspaceActions } from "pages/WorkspacePage/WorkspaceActions/WorkspaceActions";
import {
  Topbar,
  TopbarAvatar,
  TopbarData,
  TopbarDivider,
  TopbarIcon,
  TopbarIconButton,
} from "components/FullPageLayout/Topbar";
import Tooltip from "@mui/material/Tooltip";
import ArrowBackOutlined from "@mui/icons-material/ArrowBackOutlined";
import PersonOutlineOutlined from "@mui/icons-material/PersonOutlineOutlined";
import { WorkspaceOutdatedTooltipContent } from "components/WorkspaceOutdatedTooltip/WorkspaceOutdatedTooltip";
import { Popover, PopoverTrigger } from "components/Popover/Popover";
import ScheduleOutlined from "@mui/icons-material/ScheduleOutlined";
import { WorkspaceStatusBadge } from "components/WorkspaceStatusBadge/WorkspaceStatusBadge";
import { Pill } from "components/Pill/Pill";
import { WorkspaceScheduleControls } from "../WorkspaceScheduleControls";
import { DormantTopbarData } from "./DormantTopbarData";
import { workspaceQuota } from "api/queries/workspaceQuota";
import { useQuery } from "react-query";
import MonetizationOnOutlined from "@mui/icons-material/MonetizationOnOutlined";
import { useTheme } from "@mui/material/styles";
import InfoOutlined from "@mui/icons-material/InfoOutlined";
import Link from "@mui/material/Link";

export type WorkspaceError =
  | "getBuildsError"
  | "buildError"
  | "cancellationError";

export type WorkspaceErrors = Partial<Record<WorkspaceError, unknown>>;

export interface WorkspaceProps {
  handleStart: (buildParameters?: TypesGen.WorkspaceBuildParameter[]) => void;
  handleStop: () => void;
  handleRestart: (buildParameters?: TypesGen.WorkspaceBuildParameter[]) => void;
  handleDelete: () => void;
  handleUpdate: () => void;
  handleCancel: () => void;
  handleSettings: () => void;
  handleChangeVersion: () => void;
  handleDormantActivate: () => void;
  isUpdating: boolean;
  isRestarting: boolean;
  workspace: TypesGen.Workspace;
  canUpdateWorkspace: boolean;
  canChangeVersions: boolean;
  canRetryDebugMode: boolean;
  handleBuildRetry: () => void;
  handleBuildRetryDebug: () => void;
}

export const WorkspaceTopbar = (props: WorkspaceProps) => {
  const {
    handleStart,
    handleStop,
    handleRestart,
    handleDelete,
    handleUpdate,
    handleCancel,
    handleSettings,
    handleChangeVersion,
    handleDormantActivate,
    workspace,
    isUpdating,
    isRestarting,
    canUpdateWorkspace,
    canChangeVersions,
    canRetryDebugMode,
    handleBuildRetry,
    handleBuildRetryDebug,
  } = props;
  const theme = useTheme();

  // Quota
  const hasDailyCost = workspace.latest_build.daily_cost > 0;
  const { data: quota } = useQuery({
    ...workspaceQuota(workspace.owner_name),
    enabled: hasDailyCost,
  });

  return (
    <Topbar>
      <Tooltip title="Back to workspaces">
        <TopbarIconButton component={RouterLink} to="workspaces">
          <ArrowBackOutlined />
        </TopbarIconButton>
      </Tooltip>

      <div
        css={{
          paddingLeft: 16,
          display: "flex",
          alignItems: "center",
          gap: 32,
        }}
      >
        <TopbarData>
          <TopbarAvatar src={workspace.template_icon} />
          <span css={{ fontWeight: 500 }}>{workspace.name}</span>
          <TopbarDivider />
          <Link
            component={RouterLink}
            to={`/templates/${workspace.template_name}`}
            css={{ color: "inherit" }}
          >
            {workspace.template_display_name ?? workspace.template_name}
          </Link>

          {workspace.outdated ? (
            <Popover mode="hover">
              <PopoverTrigger>
                {/* Added to give some bottom space from the popover content */}
                <div css={{ padding: "4px 0" }}>
                  <Pill
                    icon={
                      <InfoOutlined
                        css={{
                          width: "12px !important",
                          height: "12px !important",
                          color: theme.palette.warning.light,
                        }}
                      />
                    }
                    text={
                      <span css={{ color: theme.palette.warning.light }}>
                        {workspace.latest_build.template_version_name}
                      </span>
                    }
                  />
                </div>
              </PopoverTrigger>
              <WorkspaceOutdatedTooltipContent
                templateName={workspace.template_name}
                latestVersionId={workspace.template_active_version_id}
                onUpdateVersion={handleUpdate}
                ariaLabel="update version"
              />
            </Popover>
          ) : (
            <Pill text={workspace.latest_build.template_version_name} />
          )}
        </TopbarData>

        <TopbarData>
          <Tooltip title="Owner">
            <TopbarIcon>
              <PersonOutlineOutlined aria-label="Owner" />
            </TopbarIcon>
          </Tooltip>
          <span>{workspace.owner_name}</span>
        </TopbarData>

        <DormantTopbarData workspace={workspace} />

        <TopbarData>
          <TopbarIcon>
            <Tooltip title="Schedule">
              <ScheduleOutlined aria-label="Schedule" />
            </Tooltip>
          </TopbarIcon>
          <WorkspaceScheduleControls
            workspace={workspace}
            canUpdateSchedule={canUpdateWorkspace}
          />
        </TopbarData>

        {quota && (
          <TopbarData>
            <TopbarIcon>
              <Tooltip title="Daily usage">
                <MonetizationOnOutlined aria-label="Daily usage" />
              </Tooltip>
            </TopbarIcon>
            <span>
              {workspace.latest_build.daily_cost}{" "}
              <span css={{ color: theme.palette.text.secondary }}>
                credits of
              </span>{" "}
              {quota.budget}
            </span>
          </TopbarData>
        )}
      </div>

      <div
        css={{
          marginLeft: "auto",
          display: "flex",
          alignItems: "center",
          gap: 12,
        }}
      >
        <WorkspaceStatusBadge workspace={workspace} />
        <WorkspaceActions
          workspace={workspace}
          handleStart={handleStart}
          handleStop={handleStop}
          handleRestart={handleRestart}
          handleDelete={handleDelete}
          handleUpdate={handleUpdate}
          handleCancel={handleCancel}
          handleSettings={handleSettings}
          handleRetry={handleBuildRetry}
          handleRetryDebug={handleBuildRetryDebug}
          handleChangeVersion={handleChangeVersion}
          handleDormantActivate={handleDormantActivate}
          canRetryDebug={canRetryDebugMode}
          canChangeVersions={canChangeVersions}
          isUpdating={isUpdating}
          isRestarting={isRestarting}
        />
      </div>
    </Topbar>
  );
};
