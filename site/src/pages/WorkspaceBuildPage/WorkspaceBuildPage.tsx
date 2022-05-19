import { makeStyles } from "@material-ui/core/styles"
import Typography from "@material-ui/core/Typography"
import { useMachine } from "@xstate/react"
import dayjs from "dayjs"
import React from "react"
import { useParams } from "react-router-dom"
import { ProvisionerJobLog } from "../../api/types"
import { Loader } from "../../components/Loader/Loader"
import { Logs } from "../../components/Logs/Logs"
import { Margins } from "../../components/Margins/Margins"
import { Stack } from "../../components/Stack/Stack"
import { WorkspaceBuildStats } from "../../components/WorkspaceBuildStats/WorkspaceBuildStats"
import { MONOSPACE_FONT_FAMILY } from "../../theme/constants"
import { workspaceBuildMachine } from "../../xServices/workspaceBuild/workspaceBuildXService"

type Stage = ProvisionerJobLog["stage"]

const groupLogsByStage = (logs: ProvisionerJobLog[]) => {
  const logsByStage: Record<Stage, ProvisionerJobLog[]> = {}

  for (const log of logs) {
    // If there is no log in the stage record, add an empty array
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
    if (logsByStage[log.stage] === undefined) {
      logsByStage[log.stage] = []
    }

    logsByStage[log.stage].push(log)
  }

  return logsByStage
}

const useBuildId = () => {
  const { buildId } = useParams()

  if (!buildId) {
    throw new Error("buildId param is required.")
  }

  return buildId
}

const getStageDurationInSeconds = (logs: ProvisionerJobLog[]) => {
  if (logs.length < 2) {
    return
  }

  const startedAt = dayjs(logs[0].created_at)
  const completedAt = dayjs(logs[logs.length - 1].created_at)
  return completedAt.diff(startedAt, "seconds")
}

export const WorkspaceBuildPage: React.FC = () => {
  const buildId = useBuildId()
  const [buildState] = useMachine(workspaceBuildMachine, { context: { buildId } })
  const { logs, build } = buildState.context
  const groupedLogsByStage = logs ? groupLogsByStage(logs) : undefined
  const stages = groupedLogsByStage ? Object.keys(groupedLogsByStage) : undefined
  const styles = useStyles()

  return (
    <Margins>
      <Stack>
        <Typography variant="h4" className={styles.title}>
          Logs
        </Typography>

        {build && <WorkspaceBuildStats build={build} />}
        {!groupedLogsByStage && <Loader />}
        {groupedLogsByStage && stages && (
          <div className={styles.logs}>
            {stages.map((stage) => {
              const logs = groupedLogsByStage[stage]
              const isEmpty = logs.every((l) => l.output === "")
              const lines = logs.map((l) => ({
                time: l.created_at,
                output: l.output,
              }))
              const duration = getStageDurationInSeconds(logs)

              return (
                <div key={stage}>
                  <div className={styles.header}>
                    <div>{stage}</div>
                    {duration && <div className={styles.duration}>{duration} seconds</div>}
                  </div>
                  {!isEmpty && <Logs lines={lines} className={styles.codeBlock} />}
                </div>
              )
            })}
          </div>
        )}
      </Stack>
    </Margins>
  )
}

const useStyles = makeStyles((theme) => ({
  title: {
    marginTop: theme.spacing(5),
  },

  logs: {
    border: `1px solid ${theme.palette.divider}`,
    borderRadius: 2,
    fontFamily: MONOSPACE_FONT_FAMILY,
  },

  header: {
    fontSize: theme.typography.body1.fontSize,
    padding: theme.spacing(2),
    paddingLeft: theme.spacing(4),
    borderBottom: `1px solid ${theme.palette.divider}`,
    backgroundColor: theme.palette.background.paper,
    display: "flex",
    alignItems: "center",
  },

  duration: {
    marginLeft: "auto",
    color: theme.palette.text.secondary,
    fontSize: theme.typography.body2.fontSize,
  },

  codeBlock: {
    padding: theme.spacing(2),
    paddingLeft: theme.spacing(4),
  },
}))
