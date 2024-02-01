// Default port from the server
export const defaultPort = 3000;
export const prometheusPort = 2114;

// Use alternate ports in case we're running in a Coder Workspace.
export const agentPProfPort = 6061;
export const coderdPProfPort = 6062;

// Credentials for the first user
export const username = "admin";
export const password = "SomeSecurePassword!";
export const email = "admin@coder.com";

export const gitAuth = {
  deviceProvider: "device",
  webProvider: "web",
  // These ports need to be hardcoded so that they can be
  // used in `playwright.config.ts` to set the environment
  // variables for the server.
  devicePort: 50515,
  webPort: 50516,

  authPath: "/auth",
  tokenPath: "/token",
  codePath: "/code",
  validatePath: "/validate",
  installationsPath: "/installations",
};
