import { ComponentMeta, Story } from "@storybook/react"
import dayjs from "dayjs"
import {
  MockProvisionerJob,
  MockStartingWorkspace,
  MockTemplate,
  MockWorkspace,
  MockWorkspaceBuild,
} from "../../testHelpers/renderHelpers"
import {
  WorkspaceBuildProgress,
  WorkspaceBuildProgressProps,
} from "./WorkspaceBuildProgress"

export default {
  title: "components/WorkspaceBuildProgress",
  component: WorkspaceBuildProgress,
} as ComponentMeta<typeof WorkspaceBuildProgress>

const Template: Story<WorkspaceBuildProgressProps> = (args) => (
  <WorkspaceBuildProgress {...args} />
)

export const Starting = Template.bind({})
Starting.args = {
  template: {
    ...MockTemplate,
    build_time_stats: {
      start_ms: 10000,
    },
  },
  workspace: {
    ...MockStartingWorkspace,
    latest_build: {
      ...MockWorkspaceBuild,
      status: "starting",
      job: {
        ...MockProvisionerJob,
        started_at: dayjs().add(-5, "second").format(),
        status: "running",
      },
    },
  },
}

export const StartingUnknown = Template.bind({})
StartingUnknown.args = {
  ...Starting.args,
  template: {
    ...MockTemplate,
    build_time_stats: {
      start_ms: undefined,
    },
  },
}

export const StartingPassedEstimate = Template.bind({})
StartingPassedEstimate.args = {
  ...Starting.args,
  template: {
    ...MockTemplate,
    build_time_stats: {
      start_ms: 1000,
    },
  },
}
