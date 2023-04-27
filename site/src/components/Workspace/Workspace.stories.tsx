import { action } from "@storybook/addon-actions"
import { Story } from "@storybook/react"
import { WatchAgentMetadataContext } from "components/Resources/AgentMetadata"
import { ProvisionerJobLog } from "api/typesGenerated"
import * as Mocks from "../../testHelpers/entities"
import { Workspace, WorkspaceErrors, WorkspaceProps } from "./Workspace"
import { withReactContext } from "storybook-react-context"
import EventSource from "eventsourcemock"
import { ProxyContext, getURLs } from "contexts/ProxyContext"

export default {
  title: "components/Workspace",
  component: Workspace,
  argTypes: {},
  decorators: [
    withReactContext({
      Context: WatchAgentMetadataContext,
      initialState: (_: string): EventSource => {
        // Need Bruno's help here.
        return new EventSource()
      },
    }),
  ],
}

const Template: Story<WorkspaceProps> = (args) => (
  <ProxyContext.Provider
    value={{
      proxy: getURLs([], undefined),
      isLoading: false,
      setProxy: () => {
        return
      },
    }}
  >
    <Workspace {...args} />
  </ProxyContext.Provider>
)

export const Running = Template.bind({})
Running.args = {
  scheduleProps: {
    onDeadlineMinus: () => {
      // do nothing, this is just for storybook
    },
    onDeadlinePlus: () => {
      // do nothing, this is just for storybook
    },
    maxDeadlineDecrease: 0,
    maxDeadlineIncrease: 24,
  },
  workspace: Mocks.MockWorkspace,
  handleStart: action("start"),
  handleStop: action("stop"),
  resources: [
    Mocks.MockWorkspaceResource,
    Mocks.MockWorkspaceResource2,
    Mocks.MockWorkspaceResource3,
  ],
  builds: [Mocks.MockWorkspaceBuild],
  canUpdateWorkspace: true,
  workspaceErrors: {},
  buildInfo: Mocks.MockBuildInfo,
  template: Mocks.MockTemplate,
}

export const WithoutUpdateAccess = Template.bind({})
WithoutUpdateAccess.args = {
  ...Running.args,
  canUpdateWorkspace: false,
}

export const Starting = Template.bind({})
Starting.args = {
  ...Running.args,
  workspace: Mocks.MockStartingWorkspace,
}

export const Stopped = Template.bind({})
Stopped.args = {
  ...Running.args,
  workspace: Mocks.MockStoppedWorkspace,
}

export const Stopping = Template.bind({})
Stopping.args = {
  ...Running.args,
  workspace: Mocks.MockStoppingWorkspace,
}

export const Failed = Template.bind({})
Failed.args = {
  ...Running.args,
  workspace: Mocks.MockFailedWorkspace,
  workspaceErrors: {
    [WorkspaceErrors.BUILD_ERROR]: Mocks.makeMockApiError({
      message: "A workspace build is already active.",
    }),
  },
}

export const FailedWithLogs = Template.bind({})
FailedWithLogs.args = {
  ...Running.args,
  workspace: {
    ...Mocks.MockFailedWorkspace,
    latest_build: {
      ...Mocks.MockFailedWorkspace.latest_build,
      job: {
        ...Mocks.MockFailedWorkspace.latest_build.job,
        error:
          "recv workspace provision: plan terraform: terraform plan: exit status 1",
      },
    },
  },
  failedBuildLogs: makeFailedBuildLogs(),
}

export const Deleting = Template.bind({})
Deleting.args = {
  ...Running.args,
  workspace: Mocks.MockDeletingWorkspace,
}

export const Deleted = Template.bind({})
Deleted.args = {
  ...Running.args,
  workspace: Mocks.MockDeletedWorkspace,
}

export const Canceling = Template.bind({})
Canceling.args = {
  ...Running.args,
  workspace: Mocks.MockCancelingWorkspace,
}

export const Canceled = Template.bind({})
Canceled.args = {
  ...Running.args,
  workspace: Mocks.MockCanceledWorkspace,
}

export const Outdated = Template.bind({})
Outdated.args = {
  ...Running.args,
  workspace: Mocks.MockOutdatedWorkspace,
}

export const GetBuildsError = Template.bind({})
GetBuildsError.args = {
  ...Running.args,
  workspaceErrors: {
    [WorkspaceErrors.GET_BUILDS_ERROR]: Mocks.makeMockApiError({
      message: "There is a problem fetching builds.",
    }),
  },
}

export const CancellationError = Template.bind({})
CancellationError.args = {
  ...Failed.args,
  workspaceErrors: {
    [WorkspaceErrors.CANCELLATION_ERROR]: Mocks.makeMockApiError({
      message: "Job could not be canceled.",
    }),
  },
}

function makeFailedBuildLogs(): ProvisionerJobLog[] {
  return [
    {
      id: 2362,
      created_at: "2023-03-21T15:57:42.637Z",
      log_source: "provisioner_daemon",
      log_level: "info",
      stage: "Setting up",
      output: "",
    },
    {
      id: 2363,
      created_at: "2023-03-21T15:57:42.674Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "",
    },
    {
      id: 2364,
      created_at: "2023-03-21T15:57:42.674Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "Initializing the backend...",
    },
    {
      id: 2365,
      created_at: "2023-03-21T15:57:42.674Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "",
    },
    {
      id: 2366,
      created_at: "2023-03-21T15:57:42.674Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "Initializing provider plugins...",
    },
    {
      id: 2367,
      created_at: "2023-03-21T15:57:42.674Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: '- Finding coder/coder versions matching "~\u003e 0.6.17"...',
    },
    {
      id: 2368,
      created_at: "2023-03-21T15:57:42.84Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output:
        '- Finding kreuzwerker/docker versions matching "~\u003e 3.0.1"...',
    },
    {
      id: 2369,
      created_at: "2023-03-21T15:57:42.986Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output:
        "- Using kreuzwerker/docker v3.0.2 from the shared cache directory",
    },
    {
      id: 2370,
      created_at: "2023-03-21T15:57:43.03Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "- Using coder/coder v0.6.20 from the shared cache directory",
    },
    {
      id: 2371,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "",
    },
    {
      id: 2372,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output:
        "Terraform has created a lock file .terraform.lock.hcl to record the provider",
    },
    {
      id: 2373,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output:
        "selections it made above. Include this file in your version control repository",
    },
    {
      id: 2374,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output:
        "so that Terraform can guarantee to make the same selections by default when",
    },
    {
      id: 2375,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: 'you run "terraform init" in the future.',
    },
    {
      id: 2376,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "",
    },
    {
      id: 2377,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "",
    },
    {
      id: 2378,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "Warning: Incomplete lock file information for providers",
    },
    {
      id: 2379,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "",
    },
    {
      id: 2380,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output:
        "Due to your customized provider installation methods, Terraform was forced to",
    },
    {
      id: 2381,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output:
        "calculate lock file checksums locally for the following providers:",
    },
    {
      id: 2382,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "  - coder/coder",
    },
    {
      id: 2383,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "  - kreuzwerker/docker",
    },
    {
      id: 2384,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "",
    },
    {
      id: 2385,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output:
        "The current .terraform.lock.hcl file only includes checksums for linux_amd64,",
    },
    {
      id: 2386,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output:
        "so Terraform running on another platform will fail to install these",
    },
    {
      id: 2387,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "providers.",
    },
    {
      id: 2388,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "",
    },
    {
      id: 2389,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "To calculate additional checksums for another platform, run:",
    },
    {
      id: 2390,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "  terraform providers lock -platform=linux_amd64",
    },
    {
      id: 2391,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "(where linux_amd64 is the platform to generate)",
    },
    {
      id: 2392,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "",
    },
    {
      id: 2393,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "Terraform has been successfully initialized!",
    },
    {
      id: 2394,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "",
    },
    {
      id: 2395,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output:
        'You may now begin working with Terraform. Try running "terraform plan" to see',
    },
    {
      id: 2396,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output:
        "any changes that are required for your infrastructure. All Terraform commands",
    },
    {
      id: 2397,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "should now work.",
    },
    {
      id: 2398,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "",
    },
    {
      id: 2399,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output:
        "If you ever set or change modules or backend configuration for Terraform,",
    },
    {
      id: 2400,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output:
        "rerun this command to reinitialize your working directory. If you forget, other",
    },
    {
      id: 2401,
      created_at: "2023-03-21T15:57:43.059Z",
      log_source: "provisioner",
      log_level: "debug",
      stage: "Planning infrastructure",
      output: "commands will detect it and remind you to do so if necessary.",
    },
    {
      id: 2402,
      created_at: "2023-03-21T15:57:43.078Z",
      log_source: "provisioner",
      log_level: "info",
      stage: "Planning infrastructure",
      output: "Terraform 1.3.4",
    },
    {
      id: 2403,
      created_at: "2023-03-21T15:57:43.401Z",
      log_source: "provisioner",
      log_level: "info",
      stage: "Planning infrastructure",
      output: "data.coder_provisioner.me: Refreshing...",
    },
    {
      id: 2404,
      created_at: "2023-03-21T15:57:43.402Z",
      log_source: "provisioner",
      log_level: "info",
      stage: "Planning infrastructure",
      output: "data.coder_workspace.me: Refreshing...",
    },
    {
      id: 2405,
      created_at: "2023-03-21T15:57:43.402Z",
      log_source: "provisioner",
      log_level: "info",
      stage: "Planning infrastructure",
      output: "data.coder_parameter.security_groups: Refreshing...",
    },
    {
      id: 2406,
      created_at: "2023-03-21T15:57:43.402Z",
      log_source: "provisioner",
      log_level: "info",
      stage: "Planning infrastructure",
      output:
        "data.coder_provisioner.me: Refresh complete after 0s [id=993f697b-3948-4d31-8377-6c86edc90a83]",
    },
    {
      id: 2407,
      created_at: "2023-03-21T15:57:43.403Z",
      log_source: "provisioner",
      log_level: "info",
      stage: "Planning infrastructure",
      output:
        "data.coder_workspace.me: Refresh complete after 0s [id=ca18ddca-14b5-4f5f-be55-7bfd2e3c2dc9]",
    },
    {
      id: 2408,
      created_at: "2023-03-21T15:57:43.403Z",
      log_source: "provisioner",
      log_level: "info",
      stage: "Planning infrastructure",
      output:
        "data.coder_parameter.security_groups: Refresh complete after 0s [id=9832a15f-267b-4abf-9c23-e4265af0befa]",
    },
    {
      id: 2409,
      created_at: "2023-03-21T15:57:43.405Z",
      log_source: "provisioner",
      log_level: "info",
      stage: "Planning infrastructure",
      output:
        "coder_agent.main: Refreshing state... [id=6c3718cb-605b-4b68-b26f-46dba8767f43]",
    },
    {
      id: 2410,
      created_at: "2023-03-21T15:57:43.406Z",
      log_source: "provisioner",
      log_level: "info",
      stage: "Planning infrastructure",
      output:
        "coder_agent.main: Refresh complete [id=6c3718cb-605b-4b68-b26f-46dba8767f43]",
    },
    {
      id: 2411,
      created_at: "2023-03-21T15:57:43.407Z",
      log_source: "provisioner",
      log_level: "info",
      stage: "Planning infrastructure",
      output:
        "docker_volume.home_volume: Refreshing state... [id=coder-ca18ddca-14b5-4f5f-be55-7bfd2e3c2dc9-home]",
    },
    {
      id: 2412,
      created_at: "2023-03-21T15:57:43.41Z",
      log_source: "provisioner",
      log_level: "info",
      stage: "Planning infrastructure",
      output:
        "coder_app.code-server: Refreshing state... [id=4a45a1cc-9861-4a9c-bd2f-3a2f1abc4c65]",
    },
    {
      id: 2413,
      created_at: "2023-03-21T15:57:43.411Z",
      log_source: "provisioner",
      log_level: "info",
      stage: "Planning infrastructure",
      output:
        "coder_app.code-server: Refresh complete [id=4a45a1cc-9861-4a9c-bd2f-3a2f1abc4c65]",
    },
    {
      id: 2414,
      created_at: "2023-03-21T15:57:43.417Z",
      log_source: "provisioner",
      log_level: "error",
      stage: "Planning infrastructure",
      output:
        "Error: Unable to inspect volume: Error: No such volume: coder-ca18ddca-14b5-4f5f-be55-7bfd2e3c2dc9-home",
    },
    {
      id: 2415,
      created_at: "2023-03-21T15:57:43.418Z",
      log_source: "provisioner",
      log_level: "error",
      stage: "Planning infrastructure",
      output: 'on main.tf line 61, in resource "docker_volume" "home_volume":',
    },
    {
      id: 2416,
      created_at: "2023-03-21T15:57:43.418Z",
      log_source: "provisioner",
      log_level: "error",
      stage: "Planning infrastructure",
      output: '  61: resource "docker_volume" "home_volume" {',
    },
    {
      id: 2417,
      created_at: "2023-03-21T15:57:43.418Z",
      log_source: "provisioner",
      log_level: "error",
      stage: "Planning infrastructure",
      output: "",
    },
    {
      id: 2418,
      created_at: "2023-03-21T15:57:43.419Z",
      log_source: "provisioner",
      log_level: "error",
      stage: "Planning infrastructure",
      output: "",
    },
    {
      id: 2419,
      created_at: "2023-03-21T15:57:43.422Z",
      log_source: "provisioner_daemon",
      log_level: "info",
      stage: "Cleaning Up",
      output: "",
    },
  ]
}
