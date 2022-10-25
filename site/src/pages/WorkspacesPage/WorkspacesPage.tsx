import { useMachine } from "@xstate/react"
import { getPaginationContext } from "components/PaginationWidget/utils"
import { FC } from "react"
import { Helmet } from "react-helmet-async"
import { useSearchParams } from "react-router-dom"
import { workspaceFilterQuery } from "util/filters"
import { pageTitle } from "util/page"
import { PaginationMachineRef } from "xServices/pagination/paginationXService"
import {
  workspacesMachine,
} from "xServices/workspaces/workspacesXService"
import { WorkspacesPageView } from "./WorkspacesPageView"

const WorkspacesPage: FC = () => {
  const [searchParams, setSearchParams] = useSearchParams()
  const filter = searchParams.get("filter") ?? workspaceFilterQuery.me
  const [workspacesState, send] = useMachine(workspacesMachine, {
    context: {
      filter,
      paginationContext: getPaginationContext(searchParams, setSearchParams)
    },
  })

  const { workspaceRefs, count, getWorkspacesError, getCountError } =
    workspacesState.context
  const paginationRef = workspacesState.context.paginationRef as PaginationMachineRef

  return (
    <>
      <Helmet>
        <title>{pageTitle("Workspaces")}</title>
      </Helmet>

      <WorkspacesPageView
        filter={workspacesState.context.filter}
        isLoading={!workspaceRefs}
        workspaceRefs={workspaceRefs}
        count={count}
        getWorkspacesError={getWorkspacesError}
        getCountError={getCountError}
        onFilter={(query) => {
          setSearchParams({ filter: query })
          send({
            type: "UPDATE_FILTER",
            query,
          })
        }}
        paginationRef={paginationRef}
      />
    </>
  )
}

export default WorkspacesPage
