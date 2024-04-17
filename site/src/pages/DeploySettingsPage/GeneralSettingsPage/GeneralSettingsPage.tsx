import type { FC } from "react";
import { Helmet } from "react-helmet-async";
import { useQuery } from "react-query";
import { deploymentDAUs } from "api/queries/deployment";
import { entitlements } from "api/queries/entitlements";
import { availableExperiments, experiments } from "api/queries/experiments";
import { pageTitle } from "utils/page";
import { useDeploySettings } from "../DeploySettingsLayout";
import { GeneralSettingsPageView } from "./GeneralSettingsPageView";

const GeneralSettingsPage: FC = () => {
  const { deploymentValues } = useDeploySettings();
  const deploymentDAUsQuery = useQuery(deploymentDAUs());
  const entitlementsQuery = useQuery(entitlements());
  const enabledExperimentsQuery = useQuery(experiments());
  const safeExperimentsQuery = useQuery(availableExperiments());

  const safeExperiments: string[] = [];
  const invalidExperiments: string[] = [];

  (enabledExperimentsQuery.data ?? []).forEach(function (exp) {
    const found = (safeExperimentsQuery.data?.safe ?? []).find((value) => {
      return exp === value;
    });

    if (found) {
      safeExperiments.push(exp);
    } else {
      invalidExperiments.push(exp);
    }
  });

  return (
    <>
      <Helmet>
        <title>{pageTitle("General Settings")}</title>
      </Helmet>
      <GeneralSettingsPageView
        deploymentOptions={deploymentValues.options}
        deploymentDAUs={deploymentDAUsQuery.data}
        deploymentDAUsError={deploymentDAUsQuery.error}
        entitlements={entitlementsQuery.data}
        invalidExperiments={invalidExperiments}
        safeExperiments={safeExperiments}
      />
    </>
  );
};

export default GeneralSettingsPage;
