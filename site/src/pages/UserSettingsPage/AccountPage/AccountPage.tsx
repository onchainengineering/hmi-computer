import type { FC } from "react";
import { useQuery } from "react-query";
import { groupsForUser } from "api/queries/groups";
import { Stack } from "components/Stack/Stack";
import { useAuth, useAuthenticated } from "contexts/auth/useAuth";
import { useDashboard } from "modules/dashboard/useDashboard";
import { Section } from "../Section";
import { AccountForm } from "./AccountForm";
import { AccountUserGroups } from "./AccountUserGroups";

export const AccountPage: FC = () => {
  const { user: me, permissions } = useAuthenticated();
  const { orgId: organizationId } = useAuthenticated();
  const { updateProfile, updateProfileError, isUpdatingProfile } = useAuth();
  const { entitlements } = useDashboard();

  const hasGroupsFeature = entitlements.features.user_role_management.enabled;
  const groupsQuery = useQuery({
    ...groupsForUser(organizationId, me.id),
    enabled: hasGroupsFeature,
  });

  return (
    <Stack spacing={6}>
      <Section title="Account" description="Update your account info">
        <AccountForm
          editable={permissions?.updateUsers ?? false}
          email={me.email}
          updateProfileError={updateProfileError}
          isLoading={isUpdatingProfile}
          initialValues={{ username: me.username, name: me.name }}
          onSubmit={updateProfile}
        />
      </Section>

      {hasGroupsFeature && (
        <AccountUserGroups
          groups={groupsQuery.data}
          loading={groupsQuery.isLoading}
          error={groupsQuery.error}
        />
      )}
    </Stack>
  );
};

export default AccountPage;
