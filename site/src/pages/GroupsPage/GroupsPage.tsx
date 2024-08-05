import { type FC, useEffect } from "react";
import { Helmet } from "react-helmet-async";
import { useQuery } from "react-query";
import { getErrorMessage } from "api/errors";
import { groups } from "api/queries/groups";
import { organizationPermissions } from "api/queries/organizations";
import { displayError } from "components/GlobalSnackbar/utils";
import { Loader } from "components/Loader/Loader";
import { useFeatureVisibility } from "modules/dashboard/useFeatureVisibility";
import { pageTitle } from "utils/page";
import GroupsPageView from "./GroupsPageView";

export const GroupsPage: FC = () => {
  const { template_rbac: isTemplateRBACEnabled } = useFeatureVisibility();
  const groupsQuery = useQuery(groups("default"));
  const permissionsQuery = useQuery(
    organizationPermissions("00000000-0000-0000-0000-00000000000"),
  );

  useEffect(() => {
    if (groupsQuery.error) {
      displayError(
        getErrorMessage(groupsQuery.error, "Unable to load groups."),
      );
    }
  }, [groupsQuery.error]);

  useEffect(() => {
    if (permissionsQuery.error) {
      displayError(
        getErrorMessage(permissionsQuery.error, "Unable to load permissions."),
      );
    }
  }, [permissionsQuery.error]);

  const permissions = permissionsQuery.data;
  if (!permissions) {
    return <Loader />;
  }

  return (
    <>
      <Helmet>
        <title>{pageTitle("Groups")}</title>
      </Helmet>

      <GroupsPageView
        groups={groupsQuery.data}
        canCreateGroup={permissions.createGroup}
        isTemplateRBACEnabled={isTemplateRBACEnabled}
      />
    </>
  );
};

export default GroupsPage;
