import { createContext, type FC, Suspense, useContext } from "react";
import { useQuery } from "react-query";
import { Outlet } from "react-router-dom";
import { deploymentConfig } from "api/queries/deployment";
import { organizations } from "api/queries/organizations";
import type { Organization } from "api/typesGenerated";
import { Loader } from "components/Loader/Loader";
import { Margins } from "components/Margins/Margins";
import { Stack } from "components/Stack/Stack";
import { useAuthenticated } from "contexts/auth/RequireAuth";
import { RequirePermission } from "contexts/auth/RequirePermission";
import { DeploySettingsContext } from "../DeploySettingsPage/DeploySettingsLayout";
import { Sidebar } from "./Sidebar";

type OrganizationSettingsContextValue = {
  organizations: Organization[] | undefined;
};

const OrganizationSettingsContext = createContext<
  OrganizationSettingsContextValue | undefined
>(undefined);

export const useOrganizationSettings = (): OrganizationSettingsContextValue => {
  const context = useContext(OrganizationSettingsContext);
  if (!context) {
    throw new Error(
      "useOrganizationSettings should be used inside of OrganizationSettingsLayout",
    );
  }
  return context;
};

/**
 * A multi-org capable settings page layout.
 *
 * If multi-org is not enabled or licensed, this is the wrong layout to use.
 * See DeploySettingsLayoutInner instead.
 */
export const ManagementSettingsLayout: FC = () => {
  const { permissions } = useAuthenticated();
  const deploymentConfigQuery = useQuery(
    // TODO: This is probably normally fine because we will not show links to
    //       pages that need this data, but if you manually visit the page you
    //       will see an endless loader when maybe we should show a "permission
    //       denied" error or at least a 404 instead.
    permissions.viewDeploymentValues ? deploymentConfig() : { enabled: false },
  );
  const organizationsQuery = useQuery(organizations());

  // The deployment settings page also contains users, audit logs, groups and
  // organizations, so this page must be visible if you can see any of these.
  const canViewDeploymentSettingsPage =
    permissions.viewDeploymentValues ||
    permissions.viewAllUsers ||
    permissions.editAnyOrganization ||
    permissions.viewAnyAuditLog;

  return (
    <RequirePermission isFeatureVisible={canViewDeploymentSettingsPage}>
      <Margins>
        <Stack css={{ padding: "48px 0" }} direction="row" spacing={6}>
          <OrganizationSettingsContext.Provider
            value={{ organizations: organizationsQuery.data }}
          >
            <Sidebar />
            <main css={{ width: "100%" }}>
              <DeploySettingsContext.Provider
                value={{
                  deploymentValues: deploymentConfigQuery.data,
                }}
              >
                <Suspense fallback={<Loader />}>
                  <Outlet />
                </Suspense>
              </DeploySettingsContext.Provider>
            </main>
          </OrganizationSettingsContext.Provider>
        </Stack>
      </Margins>
    </RequirePermission>
  );
};
