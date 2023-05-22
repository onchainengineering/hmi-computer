import { Workspace } from "api/typesGenerated"
import { displayImpendingDeletion } from "./utils"
import { useDashboard } from "components/Dashboard/DashboardProvider"
import { Alert } from "components/Alert/Alert"
import { formatDistanceToNow } from "date-fns"

export enum Count {
  Singular,
  Multiple,
}

export const ImpendingDeletionBanner = ({
  workspace,
  onDismiss,
  displayImpendingDeletionBanner,
  count = Count.Singular,
}: {
  workspace?: Workspace
  onDismiss: () => void
  displayImpendingDeletionBanner: boolean
  count?: Count,
}): JSX.Element | null => {
  const { entitlements, experiments } = useDashboard()
  const allowAdvancedScheduling =
    entitlements.features["advanced_template_scheduling"].enabled
  // This check can be removed when https://github.com/coder/coder/milestone/19
  // is merged up
  const allowWorkspaceActions = experiments.includes("workspace_actions")

    if (
      !workspace ||
          !displayImpendingDeletion(
            workspace,
            allowAdvancedScheduling,
            allowWorkspaceActions,
          ) ||
          !displayImpendingDeletionBanner
    ) {
      return null
    }

  return (
      <Alert severity="info" onDismiss={onDismiss} dismissible>
        {count === Count.Singular
          ? `This workspace has been unused for ${formatDistanceToNow(Date.parse(workspace.last_used_at))} and is scheduled for deletion. To keep it, connect via SSH or the web terminal.`
          : "You have workspaces that will be deleted soon due to inactivity. To keep these workspaces, connect to them via SSH or the web terminal."}
      </Alert>
  )
}
