import LinearProgress from "@material-ui/core/LinearProgress"
import makeStyles from "@material-ui/core/styles/makeStyles"
import { Template, Workspace } from "api/typesGenerated"
import dayjs, { Dayjs } from "dayjs"
import { FC, useEffect, useState } from "react"
import { MONOSPACE_FONT_FAMILY } from "theme/constants"

import duration from "dayjs/plugin/duration"
import { ChooseOne, Cond } from "components/Conditionals/ChooseOne"

dayjs.extend(duration)

const estimateFinish = (
  startedAt: Dayjs,
  templateAverage?: number,
): [number, string] => {
  if (templateAverage === undefined) {
    return [0, "Unknown"]
  }
  const realPercentage = dayjs().diff(startedAt) / templateAverage

  const maxPercentage = 1
  if (realPercentage > maxPercentage) {
    return [1, "Any moment now..."]
  }

  return [
    realPercentage,
    `~${Math.ceil(
      dayjs.duration((1 - realPercentage) * templateAverage).asSeconds(),
    )} seconds remaining...`,
  ]
}

export interface WorkspaceBuildProgressProps {
  workspace: Workspace
  buildEstimate?: number
}

// EstimateTransitionTime gets the build estimate for the workspace,
// if it is in a transition state.
export const EstimateTransitionTime = (
  template: Template,
  workspace: Workspace,
): [number | undefined, boolean] => {
  switch (workspace.latest_build.status) {
    case "starting":
      return [template.build_time_stats.start_ms, true]
    case "stopping":
      return [template.build_time_stats.stop_ms, true]
    case "deleting":
      return [template.build_time_stats.delete_ms, true]
    default:
      // Not in a transition state
      return [undefined, false]
  }
}

export const WorkspaceBuildProgress: FC<WorkspaceBuildProgressProps> = ({
  workspace,
  buildEstimate,
}) => {
  const styles = useStyles()
  const job = workspace.latest_build.job
  const [progressValue, setProgressValue] = useState<number | undefined>(
    undefined,
  )

  // By default workspace is updated every second, which can cause visual stutter
  // when the build estimate is a few seconds. The timer ensures no observable
  // stutter in all cases.
  useEffect(() => {
    const updateProgress = () => {
      if (job.status !== "running") {
        setProgressValue(undefined)
        return
      }
      const est = estimateFinish(dayjs(job.started_at), buildEstimate)[0] * 100
      setProgressValue(est)
    }
    // Perform initial update
    setTimeout(updateProgress, 100)
  }, [progressValue, job, buildEstimate])

  return (
    <div className={styles.stack}>
      <LinearProgress
        value={progressValue !== undefined ? progressValue : 0}
        variant={
          // There is an initial state where progressValue may be undefined
          // (e.g. the build isn't yet running). If we flicker from the
          // indeterminate bar to the determinate bar, the vigilant user
          // perceives the bar jumping from 100% to 0%.
          progressValue !== undefined || dayjs(job.started_at).diff() < 500
            ? "determinate"
            : "indeterminate"
        }
        // If a transition is set, there is a moment on new load where the
        // bar accelerates to progressValue and then rapidly decelerates, which
        // is not indicative of true progress.
        className={styles.noTransition}
      />
      <div className={styles.barHelpers}>
        <div className={styles.label}>{`Build ${job.status}`}</div>
        <div className={styles.label}>
          <ChooseOne>
            <Cond condition={job.status === "running"}>
              {estimateFinish(dayjs(job.started_at), buildEstimate)[1]}
            </Cond>
            <Cond>Unknown ETA</Cond>
          </ChooseOne>
        </div>
      </div>
    </div>
  )
}

const useStyles = makeStyles((theme) => ({
  stack: {
    paddingLeft: theme.spacing(0.2),
    paddingRight: theme.spacing(0.2),
  },
  noTransition: {
    transition: "none",
  },
  barHelpers: {
    display: "flex",
    justifyContent: "space-between",
  },
  label: {
    fontFamily: MONOSPACE_FONT_FAMILY,
    fontSize: 12,
    textTransform: "uppercase",
    display: "block",
    fontWeight: 600,
    color: theme.palette.text.secondary,
  },
}))
