import { Meta, StoryObj } from "@storybook/react";
import { WorkspaceDeleteDialog } from "./WorkspaceDeleteDialog";
import { MockWorkspace } from "testHelpers/entities";

const meta: Meta<typeof WorkspaceDeleteDialog> = {
  title: "pages/WorkspacePage/WorkspaceDeleteDialog",
  component: WorkspaceDeleteDialog,
};

export default meta;
type Story = StoryObj<typeof WorkspaceDeleteDialog>;

const args = {
  workspace: MockWorkspace,
  canUpdateTemplate: false,
  isOpen: true,
  onCancel: () => {},
  onConfirm: () => {},
};

export const NotTemplateAdmin: Story = {
  args,
};

export const TemplateAdmin: Story = {
  args: {
    ...args,
    canUpdateTemplate: true,
  },
};
