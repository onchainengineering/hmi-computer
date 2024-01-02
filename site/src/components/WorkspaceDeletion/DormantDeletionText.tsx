import { type FC } from "react";
import type { Workspace } from "api/typesGenerated";
import { displayDormantDeletion } from "utils/dormant";
import { useDashboard } from "components/Dashboard/DashboardProvider";

interface DormantDeletionTextProps {
  workspace: Workspace;
}

export const DormantDeletionText: FC<DormantDeletionTextProps> = ({
  workspace,
}) => {
  const { entitlements, experiments } = useDashboard();
  const allowAdvancedScheduling =
    entitlements.features["advanced_template_scheduling"].enabled;
  // This check can be removed when https://github.com/coder/coder/milestone/19
  // is merged up
  const allowWorkspaceActions = experiments.includes("workspace_actions");

  if (
    !displayDormantDeletion(
      workspace,
      allowAdvancedScheduling,
      allowWorkspaceActions,
    )
  ) {
    return null;
  }

  return (
    <span
      role="status"
      css={(theme) => ({
        color: theme.palette.warning.light,
        fontWeight: 600,
      })}
    >
      Impending deletion
    </span>
  );
};
