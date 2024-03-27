import { defineConfig } from "@playwright/test";
import path from "path";
import { defaultPort, coderdPProfPort, gitAuth } from "./constants";

export const port = process.env.CODER_E2E_PORT
  ? Number(process.env.CODER_E2E_PORT)
  : defaultPort;

export const wsEndpoint = process.env.CODER_E2E_WS_ENDPOINT;

const coderMain = path.join(__dirname, "../../enterprise/cmd/coder");

// This is where auth cookies are stored!
export const storageState = path.join(__dirname, ".auth.json");

const localURL = (port: number, path: string): string => {
  return `http://localhost:${port}${path}`;
};

export default defineConfig({
  projects: [
    {
      name: "testsSetup",
      testMatch: /tests.setup\.ts/,
    },
    {
      name: "tests",
      testMatch: /.*\.spec\.ts/,
      dependencies: ["testsSetup"],
      use: {
        storageState: storageState,
      },
      timeout: 60_000,
    },
    {
      name: "enterpriseSetup",
      testMatch: /enterprise.setup\.ts/,
      dependencies: ["tests"],
      use: {
        storageState: storageState,
      },
    },
    {
      name: "enterprise",
      testMatch: /.*\.enterpriseSpec\.ts/,
      dependencies: ["enterpriseSetup"],
      use: {
        storageState: storageState,
      },
      timeout: 60_000,
    },
  ],
  reporter: [["./reporter.ts"]],
  use: {
    baseURL: `http://localhost:${port}`,
    video: "retain-on-failure",
    ...(wsEndpoint
      ? {
          connectOptions: {
            wsEndpoint: wsEndpoint,
          },
        }
      : {
          launchOptions: {
            args: ["--disable-webgl"],
          },
        }),
  },
  webServer: {
    url: `http://localhost:${port}/api/v2/deployment/config`,
    command:
      `go run -tags embed ${coderMain} server ` +
      `--global-config $(mktemp -d -t e2e-XXXXXXXXXX) ` +
      `--access-url=http://localhost:${port} ` +
      `--http-address=localhost:${port} ` +
      `--in-memory --telemetry=false ` +
      `--dangerous-disable-rate-limits ` +
      `--provisioner-daemons 10 ` +
      `--provisioner-daemons-echo ` +
      `--web-terminal-renderer=dom ` +
      `--pprof-enable`,
    env: {
      ...process.env,

      // This is the test provider for git auth with devices!
      CODER_GITAUTH_0_ID: gitAuth.deviceProvider,
      CODER_GITAUTH_0_TYPE: "github",
      CODER_GITAUTH_0_CLIENT_ID: "client",
      CODER_GITAUTH_0_CLIENT_SECRET: "secret",
      CODER_GITAUTH_0_DEVICE_FLOW: "true",
      CODER_GITAUTH_0_APP_INSTALL_URL:
        "https://github.com/apps/coder/installations/new",
      CODER_GITAUTH_0_APP_INSTALLATIONS_URL: localURL(
        gitAuth.devicePort,
        gitAuth.installationsPath,
      ),
      CODER_GITAUTH_0_TOKEN_URL: localURL(
        gitAuth.devicePort,
        gitAuth.tokenPath,
      ),
      CODER_GITAUTH_0_DEVICE_CODE_URL: localURL(
        gitAuth.devicePort,
        gitAuth.codePath,
      ),
      CODER_GITAUTH_0_VALIDATE_URL: localURL(
        gitAuth.devicePort,
        gitAuth.validatePath,
      ),

      CODER_GITAUTH_1_ID: gitAuth.webProvider,
      CODER_GITAUTH_1_TYPE: "github",
      CODER_GITAUTH_1_CLIENT_ID: "client",
      CODER_GITAUTH_1_CLIENT_SECRET: "secret",
      CODER_GITAUTH_1_AUTH_URL: localURL(gitAuth.webPort, gitAuth.authPath),
      CODER_GITAUTH_1_TOKEN_URL: localURL(gitAuth.webPort, gitAuth.tokenPath),
      CODER_GITAUTH_1_DEVICE_CODE_URL: localURL(
        gitAuth.webPort,
        gitAuth.codePath,
      ),
      CODER_GITAUTH_1_VALIDATE_URL: localURL(
        gitAuth.webPort,
        gitAuth.validatePath,
      ),
      CODER_PPROF_ADDRESS: "127.0.0.1:" + coderdPProfPort,
    },
    reuseExistingServer: false,
  },
});
