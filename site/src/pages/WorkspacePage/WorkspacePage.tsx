import { useActor } from "@xstate/react"
import React, { useContext, useEffect } from "react"
import { useParams } from "react-router-dom"
import { ErrorSummary } from "../../components/ErrorSummary/ErrorSummary"
import { FullScreenLoader } from "../../components/Loader/FullScreenLoader"
import { Margins } from "../../components/Margins/Margins"
import { Stack } from "../../components/Stack/Stack"
import { Workspace } from "../../components/Workspace/Workspace"
import { firstOrItem } from "../../util/array"
import { XServiceContext } from "../../xServices/StateContext"

export const WorkspacePage: React.FC = () => {
  const { workspace: workspaceQueryParam } = useParams()
  const workspaceId = firstOrItem(workspaceQueryParam, null)

  const xServices = useContext(XServiceContext)
  const [workspaceState, workspaceSend] = useActor(xServices.workspaceXService)
  const { workspace, template, organization, getWorkspaceError, getTemplateError, getOrganizationError } =
    workspaceState.context

  /**
   * Get workspace, template, and organization on mount and whenever workspaceId changes.
   * workspaceSend should not change.
   */
  useEffect(() => {
    workspaceId && workspaceSend({ type: "GET_WORKSPACE", workspaceId })
  }, [workspaceId, workspaceSend])

  if (workspaceState.matches("error")) {
    return <ErrorSummary error={getWorkspaceError || getTemplateError || getOrganizationError} />
  } else if (!workspace || !template || !organization) {
    return <FullScreenLoader />
  } else {
    return (
      <Margins>
        <Stack spacing={4}>
          <Workspace organization={organization} template={template} workspace={workspace}
            handleStart={() => workspaceSend("START")}
            handleStop={() => workspaceSend("STOP")}
           />
        </Stack>
      </Margins>
    )
  }
}
