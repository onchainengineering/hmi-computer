import { useMachine } from "@xstate/react"
import React, { useEffect } from "react"
import { useNavigate, useParams } from "react-router-dom"
import * as TypesGen from "../../api/typesGenerated"
import { ErrorSummary } from "../../components/ErrorSummary/ErrorSummary"
import { FullScreenLoader } from "../../components/Loader/FullScreenLoader"
import {
  WorkspaceScheduleForm,
  WorkspaceScheduleFormValues,
} from "../../components/WorkspaceStats/WorkspaceScheduleForm"
import { firstOrItem } from "../../util/array"
import { workspaceSchedule } from "../../xServices/workspaceSchedule/workspaceScheduleXService"

export const formValuesToAutoStartRequest = (
  values: WorkspaceScheduleFormValues,
): TypesGen.UpdateWorkspaceAutostartRequest => {
  if (!values.startTime) {
    return {
      schedule: "",
    }
  }

  const [HH, mm] = values.startTime.split(":")

  // Note: Space after CRON_TZ if timezone is defined
  const preparedTZ = values.timezone ? `CRON_TZ=${values.timezone} ` : ""

  const makeCronString = (dow: string) => `${preparedTZ}${mm} ${HH} * * ${dow}`

  const days = [
    values.sunday,
    values.monday,
    values.tuesday,
    values.wednesday,
    values.thursday,
    values.friday,
    values.saturday,
  ]

  const isEveryDay = days.every((day) => day)

  const isMonThroughFri =
    !values.sunday &&
    values.monday &&
    values.tuesday &&
    values.wednesday &&
    values.thursday &&
    values.friday &&
    !values.saturday &&
    !values.sunday

  // Handle special cases, falling through to comma-separation
  if (isEveryDay) {
    return {
      schedule: makeCronString("1-7"),
    }
  } else if (isMonThroughFri) {
    return {
      schedule: makeCronString("1-5"),
    }
  } else {
    const dow = days.reduce((previous, current, idx) => {
      if (!current) {
        return previous
      } else {
        const prefix = previous ? "," : ""
        return previous + prefix + idx
      }
    }, "")

    return {
      schedule: makeCronString(dow),
    }
  }
}

export const formValuesToTTLRequest = (values: WorkspaceScheduleFormValues): TypesGen.UpdateWorkspaceTTLRequest => {
  return {
    // minutes to nanoseconds
    ttl: values.ttl ? values.ttl * 60 * 1000 * 1_000_000 : undefined,
  }
}

export const WorkspaceSchedulePage: React.FC = () => {
  const navigate = useNavigate()
  const { workspace: workspaceQueryParam } = useParams()
  const workspaceId = firstOrItem(workspaceQueryParam, null)
  const [scheduleState, scheduleSend] = useMachine(workspaceSchedule)
  const { formErrors, getWorkspaceError, workspace } = scheduleState.context

  // Get workspace on mount and whenever workspaceId changes.
  // scheduleSend should not change.
  useEffect(() => {
    workspaceId && scheduleSend({ type: "GET_WORKSPACE", workspaceId })
  }, [workspaceId, scheduleSend])

  if (!workspaceId) {
    navigate("/workspaces")
    return null
  } else if (scheduleState.matches("idle") || scheduleState.matches("gettingWorkspace") || !workspace) {
    return <FullScreenLoader />
  } else if (scheduleState.matches("error")) {
    return <ErrorSummary error={getWorkspaceError} retry={() => scheduleSend({ type: "GET_WORKSPACE", workspaceId })} />
  } else if (scheduleState.matches("presentForm") || scheduleState.matches("submittingSchedule")) {
    return (
      <WorkspaceScheduleForm
        fieldErrors={formErrors}
        isLoading={scheduleState.tags.has("loading")}
        onCancel={() => {
          navigate(`/workspaces/${workspaceId}`)
        }}
        onSubmit={(values) => {
          scheduleSend({
            type: "SUBMIT_SCHEDULE",
            autoStart: formValuesToAutoStartRequest(values),
            ttl: formValuesToTTLRequest(values),
          })
        }}
      />
    )
  } else if (scheduleState.matches("submitSuccess")) {
    navigate(`/workspaces/${workspaceId}`)
    return <FullScreenLoader />
  } else {
    // Theoretically impossible - log and bail
    console.error("WorkspaceSchedulePage: unknown state :: ", scheduleState)
    navigate("/")
    return null
  }
}
