import { useOutletContext } from "react-router-dom";
import {
  Header,
  HeaderTitle,
  Main,
  GridData,
  GridDataLabel,
  GridDataValue,
  HealthyDot,
} from "./Content";
import { HealthcheckReport } from "api/typesGenerated";
import { Alert } from "components/Alert/Alert";
import { Helmet } from "react-helmet-async";
import { pageTitle } from "utils/page";

export const AccessURLPage = () => {
  const healthStatus = useOutletContext<HealthcheckReport>();
  const accessUrl = healthStatus.access_url;

  return (
    <>
      <Helmet>
        <title>{pageTitle("Access URL - Health")}</title>
      </Helmet>

      <Header>
        <HeaderTitle>
          <HealthyDot
            healthy={accessUrl.healthy}
            hasWarnings={accessUrl.warnings.length > 0}
          />
          Access URL
        </HeaderTitle>
      </Header>

      <Main>
        {accessUrl.warnings.map((warning, i) => {
          return (
            <Alert key={i} severity="warning">
              {warning.message}
            </Alert>
          );
        })}

        <GridData>
          <GridDataLabel>Healthy</GridDataLabel>
          <GridDataValue>{accessUrl.healthy ? "Yes" : "No"}</GridDataValue>

          <GridDataLabel>Severity</GridDataLabel>
          <GridDataValue>{accessUrl.severity}</GridDataValue>

          <GridDataLabel>Access URL</GridDataLabel>
          <GridDataValue>{accessUrl.access_url}</GridDataValue>

          <GridDataLabel>Reachable</GridDataLabel>
          <GridDataValue>{accessUrl.reachable ? "Yes" : "No"}</GridDataValue>

          <GridDataLabel>Status Code</GridDataLabel>
          <GridDataValue>{accessUrl.status_code}</GridDataValue>
        </GridData>
      </Main>
    </>
  );
};

export default AccessURLPage;
