import { useMachine } from "@xstate/react"
import { FC } from "react"
import { Helmet } from "react-helmet"
import { useParams } from "react-router-dom"
import { pageTitle } from "../../util/page"
import { workspaceBuildMachine } from "../../xServices/workspaceBuild/workspaceBuildXService"
import { WorkspaceBuildPageView } from "./WorkspaceBuildPageView"

export const WorkspaceBuildPage: FC = () => {
  const { username, workspace: workspaceName, buildNumber } = useParams()
  const [buildState] = useMachine(workspaceBuildMachine, { context: { username, workspaceName, buildNumber } })
  const { logs, build } = buildState.context
  const isWaitingForLogs = !buildState.matches("logs.loaded")

  return (
    <>
      <Helmet>
        <title>{build ? pageTitle(`Build #${build.build_number} · ${build.workspace_name}`) : ""}</title>
      </Helmet>

      <WorkspaceBuildPageView logs={logs} build={build} isWaitingForLogs={isWaitingForLogs} />
    </>
  )
}
