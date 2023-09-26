import {
  MockTemplate,
  MockTemplateVersion,
  MockTemplateVersionFileTree,
  MockWorkspaceBuildLogs,
  MockWorkspaceContainerResource,
  MockWorkspaceExtendedBuildLogs,
  MockWorkspaceImageResource,
  MockWorkspaceResource,
  MockWorkspaceResourceMultipleAgents,
  MockWorkspaceResourceSensitive,
  MockWorkspaceVolumeResource,
} from "testHelpers/entities";
import { TemplateVersionEditor } from "./TemplateVersionEditor";
import type { Meta, StoryObj } from "@storybook/react";

const meta: Meta<typeof TemplateVersionEditor> = {
  title: "components/TemplateVersionEditor",
  component: TemplateVersionEditor,
  args: {
    template: MockTemplate,
    templateVersion: MockTemplateVersion,
    defaultFileTree: MockTemplateVersionFileTree,
  },
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof TemplateVersionEditor>;

export const Example: Story = {};

export const Logs = {
  args: {
    buildLogs: MockWorkspaceBuildLogs,
  },
};

export const Resources: Story = {
  args: {
    buildLogs: MockWorkspaceBuildLogs,
    resources: [
      MockWorkspaceResource,
      MockWorkspaceResourceSensitive,
      MockWorkspaceResourceMultipleAgents,
      MockWorkspaceVolumeResource,
      MockWorkspaceImageResource,
      MockWorkspaceContainerResource,
    ],
  },
};

export const ManyLogs = {
  args: {
    templateVersion: {
      ...MockTemplateVersion,
      job: {
        ...MockTemplateVersion.job,
        error:
          "template import provision for start: terraform plan: exit status 1",
      },
    },
    buildLogs: MockWorkspaceExtendedBuildLogs,
  },
};

export const Published = {
  args: {
    publishedVersion: MockTemplateVersion,
  },
};
