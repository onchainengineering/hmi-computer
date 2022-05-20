import { makeStyles } from "@material-ui/core/styles"
import Typography from "@material-ui/core/Typography"
import React from "react"
import * as TypesGen from "../../api/typesGenerated"
import { MONOSPACE_FONT_FAMILY } from "../../theme/constants"
import { WorkspaceStatus } from "../../util/workspace"
import { BuildsTable } from "../BuildsTable/BuildsTable"
import { Stack } from "../Stack/Stack"
import { WorkspaceActions } from "../WorkspaceActions/WorkspaceActions"
import { WorkspaceSection } from "../WorkspaceSection/WorkspaceSection"
import { WorkspaceStats } from "../WorkspaceStats/WorkspaceStats"

export interface WorkspaceProps {
  organization?: TypesGen.Organization
  workspace: TypesGen.Workspace
  template?: TypesGen.Template
  handleStart: () => void
  handleStop: () => void
  handleRetry: () => void
  handleUpdate: () => void
  workspaceStatus: WorkspaceStatus
  builds?: TypesGen.WorkspaceBuild[]
}

/**
 * Workspace is the top-level component for viewing an individual workspace
 */
export const Workspace: React.FC<WorkspaceProps> = ({
  workspace,
  handleStart,
  handleStop,
  handleRetry,
  handleUpdate,
  workspaceStatus,
  builds,
}) => {
  const styles = useStyles()

  return (
    <div className={styles.root}>
      <div className={styles.header}>
        <div>
          <Typography variant="h4" className={styles.title}>
            {workspace.name}
          </Typography>
          <Typography color="textSecondary" className={styles.subtitle}>
            {workspace.owner_name}
          </Typography>
        </div>

        <div className={styles.headerActions}>
          <WorkspaceActions
            workspace={workspace}
            handleStart={handleStart}
            handleStop={handleStop}
            handleRetry={handleRetry}
            handleUpdate={handleUpdate}
            workspaceStatus={workspaceStatus}
          />
        </div>
      </div>

      <Stack spacing={3}>
        <WorkspaceStats workspace={workspace} />
        <WorkspaceSection title="Timeline" contentsProps={{ className: styles.timelineContents }}>
          <BuildsTable builds={builds} className={styles.timelineTable} />
        </WorkspaceSection>
      </Stack>
    </div>
  )
}

export const useStyles = makeStyles((theme) => {
  return {
    root: {
      display: "flex",
      flexDirection: "column",
    },
    header: {
      paddingTop: theme.spacing(5),
      paddingBottom: theme.spacing(5),
      fontFamily: MONOSPACE_FONT_FAMILY,
      display: "flex",
      alignItems: "center",
    },
    headerActions: {
      marginLeft: "auto",
    },
    title: {
      fontWeight: 600,
      fontFamily: "inherit",
    },
    subtitle: {
      fontFamily: "inherit",
      marginTop: theme.spacing(0.5),
    },
    timelineContents: {
      margin: 0,
    },
    timelineTable: {
      border: 0,
    },
  }
})
