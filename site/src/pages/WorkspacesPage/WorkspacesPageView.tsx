import Link from "@material-ui/core/Link"
import { Maybe } from "components/Conditionals/Maybe"
import { PaginationWidget } from "components/PaginationWidget/PaginationWidget"
import { FC } from "react"
import { Link as RouterLink } from "react-router-dom"
import { Margins } from "../../components/Margins/Margins"
import {
  PageHeader,
  PageHeaderSubtitle,
  PageHeaderTitle,
} from "../../components/PageHeader/PageHeader"
import { SearchBarWithFilter } from "../../components/SearchBarWithFilter/SearchBarWithFilter"
import { Stack } from "../../components/Stack/Stack"
import { WorkspaceHelpTooltip } from "../../components/Tooltips"
import { WorkspacesTable } from "../../components/WorkspacesTable/WorkspacesTable"
import { workspaceFilterQuery } from "../../util/filters"
import { WorkspaceItemMachineRef } from "../../xServices/workspaces/workspacesXService"

export const Language = {
  pageTitle: "Workspaces",
  yourWorkspacesButton: "Your workspaces",
  allWorkspacesButton: "All workspaces",
  runningWorkspacesButton: "Running workspaces",
  createANewWorkspace: `Create a new workspace from a `,
  template: "Template",
}

export interface WorkspacesPageViewProps {
  isLoading?: boolean
  workspaceRefs?: WorkspaceItemMachineRef[]
  count?: number
  page: number
  limit: number
  filter?: string
  onFilter: (query: string) => void
  onNext: () => void
  onPrevious: () => void
  onGoToPage: (page: number) => void
}

export const WorkspacesPageView: FC<
  React.PropsWithChildren<WorkspacesPageViewProps>
> = ({
  isLoading,
  workspaceRefs,
  count,
  page,
  limit,
  filter,
  onFilter,
  onNext,
  onPrevious,
  onGoToPage,
}) => {
  const presetFilters = [
    { query: workspaceFilterQuery.me, name: Language.yourWorkspacesButton },
    { query: workspaceFilterQuery.all, name: Language.allWorkspacesButton },
    {
      query: workspaceFilterQuery.running,
      name: Language.runningWorkspacesButton,
    },
  ]

  return (
    <Margins>
      <PageHeader>
        <PageHeaderTitle>
          <Stack direction="row" spacing={1} alignItems="center">
            <span>{Language.pageTitle}</span>
            <WorkspaceHelpTooltip />
          </Stack>
        </PageHeaderTitle>

        <PageHeaderSubtitle>
          {Language.createANewWorkspace}
          <Link component={RouterLink} to="/templates">
            {Language.template}
          </Link>
          .
        </PageHeaderSubtitle>
      </PageHeader>

      <SearchBarWithFilter
        filter={filter}
        onFilter={onFilter}
        presetFilters={presetFilters}
      />

      <WorkspacesTable
        isLoading={isLoading}
        workspaceRefs={workspaceRefs}
        filter={filter}
      />

      <Maybe condition={count !== undefined && count > limit}>
        <PaginationWidget
          prevLabel=""
          nextLabel=""
          onPrevClick={onPrevious}
          onNextClick={onNext}
          onPageClick={onGoToPage}
          numRecords={count}
          activePage={page}
          numRecordsPerPage={limit}
        />
      </Maybe>
    </Margins>
  )
}
