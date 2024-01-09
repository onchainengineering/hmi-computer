import { type Interpolation, type Theme } from "@emotion/react";
import Button from "@mui/material/Button";
import AlertTitle from "@mui/material/AlertTitle";
import { type FC, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import type * as TypesGen from "api/typesGenerated";
import { Alert, AlertDetail } from "components/Alert/Alert";
import { AgentRow } from "components/Resources/AgentRow";
import { useTab } from "hooks";
import {
  ActiveTransition,
  WorkspaceBuildProgress,
} from "./WorkspaceBuildProgress";
import { WorkspaceDeletedBanner } from "./WorkspaceDeletedBanner";
import { WorkspaceTopbar } from "./WorkspaceTopbar";
import { HistorySidebar } from "./HistorySidebar";
import { dashboardContentBottomPadding, navHeight } from "theme/constants";
import { bannerHeight } from "components/Dashboard/DeploymentBanner/DeploymentBannerView";
import HistoryOutlined from "@mui/icons-material/HistoryOutlined";
import { useTheme } from "@mui/material/styles";
import { SidebarIconButton } from "components/FullPageLayout/Sidebar";
import HubOutlined from "@mui/icons-material/HubOutlined";
import { ResourcesSidebar } from "./ResourcesSidebar";
import { ResourceCard } from "components/Resources/ResourceCard";
import { WorkspaceNotifications } from "./WorkspaceNotifications";
import { WorkspacePermissions } from "./permissions";

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
  canChangeVersions: boolean;
  hideSSHButton?: boolean;
  hideVSCodeDesktopButton?: boolean;
  buildInfo?: TypesGen.BuildInfoResponse;
  sshPrefix?: string;
  template: TypesGen.Template;
  canRetryDebugMode: boolean;
  handleBuildRetry: () => void;
  handleBuildRetryDebug: () => void;
  buildLogs?: React.ReactNode;
  latestVersion?: TypesGen.TemplateVersion;
  permissions: WorkspacePermissions;
}

/**
 * Workspace is the top-level component for viewing an individual workspace
 */
export const Workspace: FC<WorkspaceProps> = ({
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
  canChangeVersions,
  hideSSHButton,
  hideVSCodeDesktopButton,
  buildInfo,
  sshPrefix,
  template,
  canRetryDebugMode,
  handleBuildRetry,
  handleBuildRetryDebug,
  buildLogs,
  latestVersion,
  permissions,
}) => {
  const navigate = useNavigate();
  const theme = useTheme();

  const transitionStats =
    template !== undefined ? ActiveTransition(template, workspace) : undefined;

  const sidebarOption = useTab("sidebar", "");
  const setSidebarOption = (newOption: string) => {
    const { set, value } = sidebarOption;
    if (value === newOption) {
      set("");
    } else {
      set(newOption);
    }
  };

  const selectedResourceId = useTab("resources", "");
  const resources = [...workspace.latest_build.resources].sort(
    (a, b) => countAgents(b) - countAgents(a),
  );
  const selectedResource = workspace.latest_build.resources.find(
    (r) => r.id === selectedResourceId.value,
  );
  useEffect(() => {
    if (resources.length > 0 && selectedResourceId.value === "") {
      selectedResourceId.set(resources[0].id);
    }
  }, [resources, selectedResourceId]);

  return (
    <div
      css={{
        flex: 1,
        display: "grid",
        gridTemplate: `
          "topbar topbar topbar" auto
          "leftbar sidebar content" 1fr / auto auto 1fr
        `,
        maxHeight: `calc(100vh - ${navHeight + bannerHeight}px)`,
        marginBottom: `-${dashboardContentBottomPadding}px`,
      }}
    >
      <WorkspaceTopbar
        workspace={workspace}
        handleStart={handleStart}
        handleStop={handleStop}
        handleRestart={handleRestart}
        handleDelete={handleDelete}
        handleUpdate={handleUpdate}
        handleCancel={handleCancel}
        handleSettings={handleSettings}
        handleBuildRetry={handleBuildRetry}
        handleBuildRetryDebug={handleBuildRetryDebug}
        handleChangeVersion={handleChangeVersion}
        handleDormantActivate={handleDormantActivate}
        canRetryDebugMode={canRetryDebugMode}
        canChangeVersions={canChangeVersions}
        isUpdating={isUpdating}
        isRestarting={isRestarting}
        canUpdateWorkspace={permissions.updateWorkspace}
      />

      <div
        css={{
          gridArea: "leftbar",
          height: "100%",
          overflowY: "auto",
          borderRight: `1px solid ${theme.palette.divider}`,
          display: "flex",
          flexDirection: "column",
        }}
      >
        <SidebarIconButton
          isActive={sidebarOption.value === "resources"}
          onClick={() => {
            setSidebarOption("resources");
          }}
        >
          <HubOutlined />
        </SidebarIconButton>
        <SidebarIconButton
          isActive={sidebarOption.value === "history"}
          onClick={() => {
            setSidebarOption("history");
          }}
        >
          <HistoryOutlined />
        </SidebarIconButton>
      </div>

      {sidebarOption.value === "resources" && (
        <ResourcesSidebar
          failed={workspace.latest_build.status === "failed"}
          resources={resources}
          selected={selectedResourceId.value}
          onChange={selectedResourceId.set}
        />
      )}
      {sidebarOption.value === "history" && (
        <HistorySidebar workspace={workspace} />
      )}

      <div css={styles.content}>
        <div css={styles.dotBackground}>
          <div css={{ display: "flex", flexDirection: "column", gap: 24 }}>
            <WorkspaceNotifications
              workspace={workspace}
              template={template}
              latestVersion={latestVersion}
              permissions={permissions}
              onRestartWorkspace={handleRestart}
            />

            {workspace.latest_build.status === "deleted" && (
              <WorkspaceDeletedBanner
                handleClick={() => navigate(`/templates`)}
              />
            )}

            {workspace.latest_build.job.error && (
              <Alert
                severity="error"
                actions={
                  <Button
                    onClick={
                      canRetryDebugMode
                        ? handleBuildRetryDebug
                        : handleBuildRetry
                    }
                    variant="text"
                    size="small"
                  >
                    Retry{canRetryDebugMode && " in debug mode"}
                  </Button>
                }
              >
                <AlertTitle>Workspace build failed</AlertTitle>
                <AlertDetail>{workspace.latest_build.job.error}</AlertDetail>
              </Alert>
            )}

            {transitionStats !== undefined && (
              <WorkspaceBuildProgress
                workspace={workspace}
                transitionStats={transitionStats}
              />
            )}

            {buildLogs}

            {selectedResource && (
              <ResourceCard
                resource={selectedResource}
                agentRow={(agent) => (
                  <AgentRow
                    key={agent.id}
                    agent={agent}
                    workspace={workspace}
                    sshPrefix={sshPrefix}
                    showApps={permissions.updateWorkspace}
                    showBuiltinApps={permissions.updateWorkspace}
                    hideSSHButton={hideSSHButton}
                    hideVSCodeDesktopButton={hideVSCodeDesktopButton}
                    serverVersion={buildInfo?.version || ""}
                    serverAPIVersion={buildInfo?.agent_api_version || ""}
                    onUpdateAgent={handleUpdate} // On updating the workspace the agent version is also updated
                  />
                )}
              />
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

const countAgents = (resource: TypesGen.WorkspaceResource) => {
  return resource.agents ? resource.agents.length : 0;
};

const styles = {
  content: {
    padding: 24,
    gridArea: "content",
    overflowY: "auto",
  },

  dotBackground: (theme) => ({
    minHeight: "100%",
    padding: 23,
    "--d": "1px",
    background: `
      radial-gradient(
        circle at
          var(--d)
          var(--d),

        ${theme.palette.text.secondary} calc(var(--d) - 1px),
        ${theme.palette.background.default} var(--d)
      )
      0 0 / 24px 24px
    `,
  }),

  actions: (theme) => ({
    [theme.breakpoints.down("md")]: {
      flexDirection: "column",
    },
  }),
} satisfies Record<string, Interpolation<Theme>>;
