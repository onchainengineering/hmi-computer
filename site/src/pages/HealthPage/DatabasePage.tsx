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

export const DatabasePage = () => {
  const healthStatus = useOutletContext<HealthcheckReport>();
  const database = healthStatus.database;

  return (
    <>
      <Helmet>
        <title>{pageTitle("Database - Health")}</title>
      </Helmet>

      <Header>
        <HeaderTitle>
          <HealthyDot
            healthy={database.healthy}
            hasWarnings={database.warnings.length > 0}
          />
          Database
        </HeaderTitle>
      </Header>

      <Main>
        {database.warnings.map((warning, i) => {
          return (
            <Alert key={i} severity="warning">
              {warning.message}
            </Alert>
          );
        })}

        <GridData>
          <GridDataLabel>Healthy</GridDataLabel>
          <GridDataValue>{database.healthy ? "Yes" : "No"}</GridDataValue>

          <GridDataLabel>Reachable</GridDataLabel>
          <GridDataValue>{database.reachable ? "Yes" : "No"}</GridDataValue>

          <GridDataLabel>Latency</GridDataLabel>
          <GridDataValue>{database.latency_ms}ms</GridDataValue>

          <GridDataLabel>Threshold</GridDataLabel>
          <GridDataValue>{database.threshold_ms}ms</GridDataValue>
        </GridData>
      </Main>
    </>
  );
};

export default DatabasePage;
