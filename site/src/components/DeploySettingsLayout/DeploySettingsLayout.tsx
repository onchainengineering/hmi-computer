import { makeStyles } from "@mui/styles";
import { Margins } from "components/Margins/Margins";
import { Stack } from "components/Stack/Stack";
import { Sidebar } from "./Sidebar";
import { createContext, Suspense, useContext, FC } from "react";
import { Loader } from "components/Loader/Loader";
import { DAUsResponse } from "api/typesGenerated";
import { RequirePermission } from "components/RequirePermission/RequirePermission";
import { usePermissions } from "hooks/usePermissions";
import { Outlet } from "react-router-dom";
import { DeploymentConfig } from "api/api";
import { useQuery } from "@tanstack/react-query";
import { deploymentConfig, deploymentDAUs } from "api/queries/deployment";

type DeploySettingsContextValue = {
  deploymentValues: DeploymentConfig;
  getDeploymentValuesError: unknown;
  deploymentDAUs?: DAUsResponse;
  getDeploymentDAUsError: unknown;
};

const DeploySettingsContext = createContext<
  DeploySettingsContextValue | undefined
>(undefined);

export const useDeploySettings = (): DeploySettingsContextValue => {
  const context = useContext(DeploySettingsContext);
  if (!context) {
    throw new Error(
      "useDeploySettings should be used inside of DeploySettingsLayout",
    );
  }
  return context;
};

export const DeploySettingsLayout: FC = () => {
  const deploymentConfigQuery = useQuery(deploymentConfig());
  const deploymentDAUsQuery = useQuery(deploymentDAUs());
  const styles = useStyles();
  const permissions = usePermissions();

  return (
    <RequirePermission isFeatureVisible={permissions.viewDeploymentValues}>
      <Margins>
        <Stack className={styles.wrapper} direction="row" spacing={6}>
          <Sidebar />
          <main className={styles.content}>
            {deploymentConfigQuery.data && deploymentDAUsQuery.data ? (
              <DeploySettingsContext.Provider
                value={{
                  deploymentValues: deploymentConfigQuery.data,
                  getDeploymentValuesError: deploymentConfigQuery.error,
                  deploymentDAUs: deploymentDAUsQuery.data,
                  getDeploymentDAUsError: deploymentDAUsQuery.error,
                }}
              >
                <Suspense fallback={<Loader />}>
                  <Outlet />
                </Suspense>
              </DeploySettingsContext.Provider>
            ) : (
              <Loader />
            )}
          </main>
        </Stack>
      </Margins>
    </RequirePermission>
  );
};

const useStyles = makeStyles((theme) => ({
  wrapper: {
    padding: theme.spacing(6, 0),
  },

  content: {
    maxWidth: 800,
    width: "100%",
  },
}));
