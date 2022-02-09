import { Box, Paper, Typography, Link as MuiLink } from "@material-ui/core"
import { makeStyles } from "@material-ui/core/styles"
import CloudCircleIcon from "@material-ui/icons/CloudCircle"
import Link from "next/link"
import React from "react"

import * as API from "../../api"

export interface WorkspaceProps {
  workspace: API.Workspace
}

namespace Constants {
  export const TitleIconSize = 48
  export const CardRadius = 8
  export const CardPadding = 20
}

/**
 * Workspace is the top-level component for viewing an individual workspace
 */
export const Workspace: React.FC<WorkspaceProps> = ({ workspace }) => {
  const styles = useStyles()
  return <div className={styles.root}>
    <Paper elevation={0} className={styles.section}>
      <div className={styles.horizontal}>
        <WorkspaceHeroIcon />
        <div className={styles.vertical}>
          <Typography variant="h4">{workspace.name}</Typography>
          <Typography variant="body2" color="textSecondary">
            <Link href="javascript:;">{workspace.project_id}</Link>
          </Typography>
        </div>
      </div>
    </Paper>
  </div>
}

// Component to render the 'Hero Icon' in the header of a workspace
export const WorkspaceHeroIcon = () => {
  return <Box mr={"1em"}>
    <CloudCircleIcon width={Constants.TitleIconSize} height={Constants.TitleIconSize} />
  </Box>
}

export const useStyles = makeStyles((theme) => {
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
    section: {
      border: `1px solid ${theme.palette.divider}`,
      borderRadius: Constants.CardRadius,
      padding: Constants.CardPadding,
    },
    icon: {
      width: Constants.TitleIconSize,
      height: Constants.TitleIconSize,
    }
  }
})