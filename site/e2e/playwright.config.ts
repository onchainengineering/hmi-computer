import * as path from "path"
import { PlaywrightTestConfig } from "@playwright/test"

const config: PlaywrightTestConfig = {
  testDir: "tests",
  globalSetup: require.resolve('./globalSetup'),

  // Create junit report file for upload to DataDog
  reporter: [['junit', { outputFile: 'junit.xml' }]],

  use: {
    baseURL: "http://localhost:3000",
    video: "retain-on-failure",
  },

  // `webServer` tells Playwright to launch a test server - more details here:
  // https://playwright.dev/docs/test-advanced#launching-a-development-web-server-during-the-tests
  webServer: {
    // Run the 'coderd' binary directly
    command: path.join(__dirname, "../../bin/coderd"),
    port: 3000,
    timeout: 120 * 10000,
    reuseExistingServer: false,
  },
}

export default config
