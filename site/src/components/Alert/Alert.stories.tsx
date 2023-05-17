import { Alert } from "./Alert"
import Button from "@mui/material/Button"
import Link from "@mui/material/Link"
import type { Meta, StoryObj } from "@storybook/react"

const meta: Meta<typeof Alert> = {
  title: "components/Alert",
  component: Alert,
}

export default meta
type Story = StoryObj<typeof Alert>

const ExampleAction = (
  <Button onClick={() => null} size="small">
    Button
  </Button>
)

export const Warning: Story = {
  args: {
    children: "This is a warning",
    severity: "warning",
  },
}

export const ErrorWithDefaultMessage: Story = {
  args: {
    children: "This is an error",
    severity: "error",
  },
}

export const WarningWithDismiss: Story = {
  args: {
    children: "This is a warning",
    dismissible: true,
    severity: "warning",
  },
}

export const WarningWithAction: Story = {
  args: {
    children: "This is a warning",
    actions: [ExampleAction],
    severity: "warning",
  },
}

export const WarningWithActionAndDismiss: Story = {
  args: {
    children: "This is a warning",
    actions: [ExampleAction],
    dismissible: true,
    severity: "warning",
  },
}

export const WithChildren: Story = {
  args: {
    children: (
      <div>
        This is a message with a <Link href="#">link</Link>
      </div>
    ),
  },
}
