import { ComponentMeta, Story } from "@storybook/react"
import React from "react"
import { ConfirmDialog, ConfirmDialogProps } from "./ConfirmDialog"

export default {
  title: "Components/Dialogs/ConfirmDialog",
  component: ConfirmDialog,
  argTypes: {
    onClose: {
      action: "onClose",
    },
    onConfirm: {
      action: "onConfirm",
    },
    open: {
      control: "boolean",
      defaultValue: true,
    },
    title: {
      defaultValue: "Confirm Dialog",
    },
  },
} as ComponentMeta<typeof ConfirmDialog>

const Template: Story<ConfirmDialogProps> = (args) => <ConfirmDialog {...args} />

export const DeleteDialog = Template.bind({})
DeleteDialog.args = {
  description: "Do you really want to delete me?",
  hideCancel: false,
  type: "delete",
}

export const InfoDialog = Template.bind({})
InfoDialog.args = {
  description: "Information is cool!",
  hideCancel: true,
  type: "info",
}
