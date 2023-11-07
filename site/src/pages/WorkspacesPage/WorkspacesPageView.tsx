import { Template, Workspace } from "api/typesGenerated";
import { PaginationWidgetBase } from "components/PaginationWidget/PaginationWidgetBase";
import { ComponentProps } from "react";
import { Margins } from "components/Margins/Margins";
import { PageHeader, PageHeaderTitle } from "components/PageHeader/PageHeader";
import { Stack } from "components/Stack/Stack";
import { WorkspaceHelpTooltip } from "./WorkspaceHelpTooltip";
import { WorkspacesTable } from "pages/WorkspacesPage/WorkspacesTable";
import { useLocalStorage } from "hooks";
import { DormantWorkspaceBanner, Count } from "components/WorkspaceDeletion";
import { ErrorAlert } from "components/Alert/ErrorAlert";
import { WorkspacesFilter } from "./filter/filter";
import { hasError, isApiValidationError } from "api/errors";
import {
  PaginationStatus,
  TableToolbar,
} from "components/TableToolbar/TableToolbar";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import DeleteOutlined from "@mui/icons-material/DeleteOutlined";
import { WorkspacesButton } from "./WorkspacesButton";
import { UseQueryResult } from "react-query";
import StopOutlined from "@mui/icons-material/StopOutlined";
import PlayArrowOutlined from "@mui/icons-material/PlayArrowOutlined";
import {
  MoreMenu,
  MoreMenuContent,
  MoreMenuItem,
  MoreMenuTrigger,
} from "components/MoreMenu/MoreMenu";
import {
  ArrowDropDown,
  ArrowDropDownOutlined,
  KeyboardArrowDownOutlined,
} from "@mui/icons-material";
import { Divider } from "@mui/material";

export const Language = {
  pageTitle: "Workspaces",
  yourWorkspacesButton: "Your workspaces",
  allWorkspacesButton: "All workspaces",
  runningWorkspacesButton: "Running workspaces",
  createWorkspace: <>Create Workspace&hellip;</>,
  seeAllTemplates: "See all templates",
  template: "Template",
};

type TemplateQuery = UseQueryResult<Template[]>;

export interface WorkspacesPageViewProps {
  error: unknown;
  workspaces?: Workspace[];
  dormantWorkspaces?: Workspace[];
  checkedWorkspaces: Workspace[];
  count?: number;
  filterProps: ComponentProps<typeof WorkspacesFilter>;
  page: number;
  limit: number;
  onPageChange: (page: number) => void;
  onUpdateWorkspace: (workspace: Workspace) => void;
  onCheckChange: (checkedWorkspaces: Workspace[]) => void;
  onDeleteAll: () => void;
  onStartAll: () => void;
  onStopAll: () => void;
  canCheckWorkspaces: boolean;
  templatesFetchStatus: TemplateQuery["status"];
  templates: TemplateQuery["data"];
  canCreateTemplate: boolean;
}

export const WorkspacesPageView = ({
  workspaces,
  dormantWorkspaces,
  error,
  limit,
  count,
  filterProps,
  onPageChange,
  onUpdateWorkspace,
  page,
  checkedWorkspaces,
  onCheckChange,
  onDeleteAll,
  onStopAll,
  onStartAll,
  canCheckWorkspaces,
  templates,
  templatesFetchStatus,
  canCreateTemplate,
}: WorkspacesPageViewProps) => {
  const { saveLocal } = useLocalStorage();

  const workspacesDeletionScheduled = dormantWorkspaces
    ?.filter((workspace) => workspace.deleting_at)
    .map((workspace) => workspace.id);

  const hasDormantWorkspace =
    dormantWorkspaces !== undefined && dormantWorkspaces.length > 0;

  return (
    <Margins>
      <PageHeader
        actions={
          <WorkspacesButton
            templates={templates}
            templatesFetchStatus={templatesFetchStatus}
          >
            {Language.createWorkspace}
          </WorkspacesButton>
        }
      >
        <PageHeaderTitle>
          <Stack direction="row" spacing={1} alignItems="center">
            <span>{Language.pageTitle}</span>
            <WorkspaceHelpTooltip />
          </Stack>
        </PageHeaderTitle>
      </PageHeader>

      <Stack>
        {hasError(error) && !isApiValidationError(error) && (
          <ErrorAlert error={error} />
        )}
        {/* <DormantWorkspaceBanner/> determines its own visibility */}
        <DormantWorkspaceBanner
          workspaces={dormantWorkspaces}
          shouldRedisplayBanner={hasDormantWorkspace}
          onDismiss={() =>
            saveLocal(
              "dismissedWorkspaceList",
              JSON.stringify(workspacesDeletionScheduled),
            )
          }
          count={Count.Multiple}
        />

        <WorkspacesFilter error={error} {...filterProps} />
      </Stack>

      <TableToolbar>
        {checkedWorkspaces.length > 0 ? (
          <>
            <Box>
              Selected <strong>{checkedWorkspaces.length}</strong> of{" "}
              <strong>{workspaces?.length}</strong>{" "}
              {workspaces?.length === 1 ? "workspace" : "workspaces"}
            </Box>

            <MoreMenu>
              <MoreMenuTrigger>
                <Button
                  variant="text"
                  size="small"
                  css={{ borderRadius: 9999, marginLeft: "auto" }}
                  endIcon={<KeyboardArrowDownOutlined />}
                >
                  Actions
                </Button>
              </MoreMenuTrigger>
              <MoreMenuContent>
                <MoreMenuItem
                  disabled={
                    !checkedWorkspaces?.every(
                      (w) => w.latest_build.status === "stopped",
                    )
                  }
                >
                  <PlayArrowOutlined /> Start
                </MoreMenuItem>
                <MoreMenuItem
                  disabled={
                    !checkedWorkspaces?.every(
                      (w) => w.latest_build.status === "running",
                    )
                  }
                >
                  <StopOutlined /> Stop
                </MoreMenuItem>
                <Divider />
                <MoreMenuItem danger onClick={onDeleteAll}>
                  <DeleteOutlined /> Delete
                </MoreMenuItem>
              </MoreMenuContent>
            </MoreMenu>
            {/* <div
              css={{
                marginLeft: "auto",
                display: "flex",
                gap: 8,
                alignItems: "center",
              }}
            >
              <Button
                disabled={
                  !workspaces?.every((w) => w.latest_build.status === "stopped")
                }
                variant="text"
                size="small"
                // Needs some manual adjustment to align with the delete button icon
                startIcon={
                  <PlayArrowOutlined css={{ width: 16, height: 16 }} />
                }
                onClick={onStartAll}
              >
                Start
              </Button>
              <div
                css={(theme) => ({
                  width: 1,
                  height: 12,
                  backgroundColor: theme.palette.divider,
                })}
              />
              <Button
                variant="text"
                disabled={
                  !workspaces?.every((w) => w.latest_build.status === "running")
                }
                size="small"
                // Needs some manual adjustment to align with the delete button icon
                startIcon={<StopOutlined css={{ width: 16, height: 16 }} />}
                onClick={onStopAll}
              >
                Stop
              </Button>
              <div
                css={(theme) => ({
                  width: 1,
                  height: 12,
                  backgroundColor: theme.palette.divider,
                })}
              />
              <Button
                variant="text"
                size="small"
                startIcon={<DeleteOutlined />}
                onClick={onDeleteAll}
                color="error"
              >
                Delete
              </Button>
            </div> */}
          </>
        ) : (
          <PaginationStatus
            isLoading={!workspaces && !error}
            showing={workspaces?.length ?? 0}
            total={count ?? 0}
            label="workspaces"
          />
        )}
      </TableToolbar>

      <WorkspacesTable
        canCreateTemplate={canCreateTemplate}
        workspaces={workspaces}
        isUsingFilter={filterProps.filter.used}
        onUpdateWorkspace={onUpdateWorkspace}
        checkedWorkspaces={checkedWorkspaces}
        onCheckChange={onCheckChange}
        canCheckWorkspaces={canCheckWorkspaces}
        templates={templates}
      />

      {count !== undefined && (
        <PaginationWidgetBase
          count={count}
          limit={limit}
          onChange={onPageChange}
          page={page}
        />
      )}
    </Margins>
  );
};
