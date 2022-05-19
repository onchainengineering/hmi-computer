import { ComponentMeta, Story } from "@storybook/react"
import React from "react"
import { MockWorkspaceBuildLogs } from "../../testHelpers/entities"
import { Logs, LogsProps } from "./Logs"

export default {
  title: "components/Logs",
  component: Logs,
} as ComponentMeta<typeof Logs>

const Template: Story<LogsProps> = (args) => <Logs {...args} />

const lines = MockWorkspaceBuildLogs.map((l) => ({
  time: l.created_at,
  output: l.output,
}))
export const Example = Template.bind({})
Example.args = {
  lines,
}
