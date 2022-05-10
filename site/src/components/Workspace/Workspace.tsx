import { makeStyles } from "@material-ui/core/styles"
import Typography from "@material-ui/core/Typography"
import React from "react"
import * as Types from "../../api/types"
import { WorkspaceStatus } from "../../pages/WorkspacePage/WorkspacePage"
import { WorkspaceSchedule } from "../WorkspaceSchedule/WorkspaceSchedule"
import { WorkspaceSection } from "../WorkspaceSection/WorkspaceSection"
import { WorkspaceStatusBar } from "../WorkspaceStatusBar/WorkspaceStatusBar"

export interface WorkspaceProps {
  organization?: Types.Organization
  workspace: Types.Workspace
  template?: Types.Template
  handleStart: () => void
  handleStop: () => void
  handleRetry: () => void
  workspaceStatus: WorkspaceStatus
}

/**
 * Workspace is the top-level component for viewing an individual workspace
 */
export const Workspace: React.FC<WorkspaceProps> = ({
  organization,
  template,
  workspace,
  handleStart,
  handleStop,
  handleRetry,
  workspaceStatus,
}) => {
  const styles = useStyles()

  return (
    <div className={styles.root}>
      <div className={styles.vertical}>
        <WorkspaceStatusBar
          organization={organization}
          template={template}
          workspace={workspace}
          handleStart={handleStart}
          handleStop={handleStop}
          handleRetry={handleRetry}
          workspaceStatus={workspaceStatus}
        />
        <div className={styles.horizontal}>
          <div className={styles.sidebarContainer}>
            <WorkspaceSection title="Applications">
              <Placeholder />
            </WorkspaceSection>
            <WorkspaceSchedule autostart={workspace.autostart_schedule} autostop={workspace.autostop_schedule} />
            <WorkspaceSection title="Dev URLs">
              <Placeholder />
            </WorkspaceSection>
            <WorkspaceSection title="Resources">
              <Placeholder />
            </WorkspaceSection>
          </div>
          <div className={styles.timelineContainer}>
            <WorkspaceSection title="Timeline">
              <div
                className={styles.vertical}
                style={{ justifyContent: "center", alignItems: "center", height: "300px" }}
              >
                <Placeholder />
              </div>
            </WorkspaceSection>
          </div>
        </div>
      </div>
    </div>
  )
}

/**
 * Temporary placeholder component until we have the sections implemented
 * Can be removed once the Workspace page has all the necessary sections
 */
const Placeholder: React.FC = () => {
  return (
    <div style={{ textAlign: "center", opacity: "0.5" }}>
      <Typography variant="caption">Not yet implemented</Typography>
    </div>
  )
}

export const useStyles = makeStyles(() => {
  return {
    root: {
      display: "flex",
      flexDirection: "column",
    },
    horizontal: {
      display: "flex",
      flexDirection: "row",
    },
    vertical: {
      display: "flex",
      flexDirection: "column",
    },
    sidebarContainer: {
      display: "flex",
      flexDirection: "column",
      flex: "0 0 350px",
    },
    timelineContainer: {
      flex: 1,
    },
  }
})
