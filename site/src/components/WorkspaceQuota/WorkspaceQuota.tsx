import Box from "@material-ui/core/Box"
import LinearProgress from "@material-ui/core/LinearProgress"
import { makeStyles } from "@material-ui/core/styles"
import Skeleton from "@material-ui/lab/Skeleton"
import { Stack } from "components/Stack/Stack"
import { FC } from "react"
import { MONOSPACE_FONT_FAMILY } from "../../theme/constants"
import * as TypesGen from "../../api/typesGenerated"

export const Language = {
  of: "of",
  workspace: "workspace",
  workspaces: "workspaces",
}

export interface WorkspaceQuotaProps {
  quota?: TypesGen.UserWorkspaceQuota
}

export const WorkspaceQuota: FC<WorkspaceQuotaProps> = ({ quota }) => {
  const styles = useStyles()

  // loading state
  if (quota === undefined) {
    return (
      <Box>
        <Stack spacing={1} className={styles.stack}>
          <span className={styles.title}>
            Workspace Quota
          </span>
          <LinearProgress color="primary" />
          <div className={styles.label}>
            <Skeleton className={styles.skeleton} />
          </div>
        </Stack>
      </Box>
    )
  }

  // don't show if limit is 0, this means the feature is disabled.
  if (quota.limit === 0) {
    return (<></>)
  }

  let value = Math.round((quota.count / quota.limit) * 100)
  // we don't want to round down to zero if the count is > 0
  if (quota.count > 0 && value === 0) {
    value = 1
  }


  return (
    <Box>
      <Stack spacing={1} className={styles.stack}>
        <span className={styles.title}>
          Workspace Quota
        </span>
        <LinearProgress className={quota.count >= quota.limit ? styles.maxProgress : undefined} value={value} variant="determinate" />
        <div className={styles.label}>
          {quota.count} {Language.of} {quota.limit}{" "}
          {quota.limit === 1 ? Language.workspace : Language.workspaces}{" used"}
        </div>
      </Stack>
    </Box>
  )
}

const useStyles = makeStyles((theme) => ({
  stack: {
    paddingTop: theme.spacing(2.5),
  },
  maxProgress: {
    "& .MuiLinearProgress-colorPrimary": {
      backgroundColor: theme.palette.error.main,
    },
    "& .MuiLinearProgress-barColorPrimary": {
      backgroundColor: theme.palette.error.main,
    },
  },
  title: {
    fontFamily: MONOSPACE_FONT_FAMILY,
    fontSize: 21,
    paddingBottom: "8px",
  },
  label: {
    fontFamily: MONOSPACE_FONT_FAMILY,
    fontSize: 12,
    textTransform: "uppercase",
    display: "block",
    fontWeight: 600,
    color: theme.palette.text.secondary,
  },
  skeleton: {
    minWidth: "150px",
  },
}))
