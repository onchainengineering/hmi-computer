import { makeStyles } from "@mui/styles"
import dayjs from "dayjs"
import { ComponentProps, FC, Fragment } from "react"
import { ProvisionerJobLog } from "../../api/typesGenerated"
import { MONOSPACE_FONT_FAMILY } from "../../theme/constants"
import { Logs } from "../Logs/Logs"
import Box from "@mui/material/Box"

const Language = {
  seconds: "seconds",
}

type Stage = ProvisionerJobLog["stage"]
type LogsGroupedByStage = Record<Stage, ProvisionerJobLog[]>
type GroupLogsByStageFn = (logs: ProvisionerJobLog[]) => LogsGroupedByStage

export const groupLogsByStage: GroupLogsByStageFn = (logs) => {
  const logsByStage: LogsGroupedByStage = {}

  for (const log of logs) {
    if (log.stage in logsByStage) {
      logsByStage[log.stage].push(log)
    } else {
      logsByStage[log.stage] = [log]
    }
  }

  return logsByStage
}

const getStageDurationInSeconds = (logs: ProvisionerJobLog[]) => {
  if (logs.length < 2) {
    return
  }

  const startedAt = dayjs(logs[0].created_at)
  const completedAt = dayjs(logs[logs.length - 1].created_at)
  return completedAt.diff(startedAt, "seconds")
}

export type WorkspaceBuildLogsProps = {
  logs: ProvisionerJobLog[]
  hideTimestamps?: boolean
} & ComponentProps<typeof Box>

export const WorkspaceBuildLogs: FC<WorkspaceBuildLogsProps> = ({
  hideTimestamps,
  logs,
  ...boxProps
}) => {
  const groupedLogsByStage = groupLogsByStage(logs)
  const stages = Object.keys(groupedLogsByStage)
  const styles = useStyles()

  return (
    <Box
      {...boxProps}
      sx={{
        ...(theme) => ({
          border: `1px solid ${theme.palette.divider}`,
          borderRadius: theme.shape.borderRadius,
          fontFamily: MONOSPACE_FONT_FAMILY,
        }),
        ...boxProps.sx,
      }}
    >
      {stages.map((stage) => {
        const logs = groupedLogsByStage[stage]
        const isEmpty = logs.every((log) => log.output === "")
        const lines = logs.map((log) => ({
          time: log.created_at,
          output: log.output,
          level: log.log_level,
        }))
        const duration = getStageDurationInSeconds(logs)
        const shouldDisplayDuration = duration !== undefined

        return (
          <Fragment key={stage}>
            <div className={styles.header}>
              <div>{stage}</div>
              {shouldDisplayDuration && (
                <div className={styles.duration}>
                  {duration} {Language.seconds}
                </div>
              )}
            </div>
            {!isEmpty && <Logs hideTimestamps={hideTimestamps} lines={lines} />}
          </Fragment>
        )
      })}
    </Box>
  )
}

const useStyles = makeStyles((theme) => ({
  header: {
    fontSize: 13,
    fontWeight: 600,
    padding: theme.spacing(0.5, 3),
    display: "flex",
    alignItems: "center",
    fontFamily: "Inter",
    borderBottom: `1px solid ${theme.palette.divider}`,
    position: "sticky",
    top: 0,
    background: theme.palette.background.default,

    "&:last-child": {
      borderBottom: 0,
      borderRadius: "0 0 8px 8px",
    },
  },

  duration: {
    marginLeft: "auto",
    color: theme.palette.text.secondary,
    fontSize: 12,
  },
}))
