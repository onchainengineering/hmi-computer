import { StoryObj } from "@storybook/react";
import { AccessURLPage } from "./AccessURLPage";
import { generateMeta } from "./storybook";
import { HEALTH_QUERY_KEY } from "api/queries/debug";
import { MockHealth } from "testHelpers/entities";
import { HealthcheckReport } from "api/typesGenerated";

const meta = {
  title: "pages/Health/AccessURL",
  ...generateMeta({
    path: "/health/access-url",
    element: <AccessURLPage />,
  }),
};

export default meta;
type Story = StoryObj;

const Example: Story = {};

const settingsWithError: HealthcheckReport = {
  ...MockHealth,
  severity: "error",
  access_url: {
    ...MockHealth.access_url,
    severity: "error",
    error:
      'EACS03: get healthz endpoint: Get "https://localhost:7080/healthz": http: server gave HTTP response to HTTPS client',
  },
};

export const WithError: Story = {
  parameters: {
    queries: [
      ...meta.parameters.queries,
      {
        key: HEALTH_QUERY_KEY,
        data: settingsWithError,
      },
    ],
  },
};

export { Example as AccessURL };
