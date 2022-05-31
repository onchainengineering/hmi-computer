import Box from "@material-ui/core/Box"
import { Story } from "@storybook/react"
import { MockUser } from "../../testHelpers/entities"
import { UserDropdown, UserDropdownProps } from "./UsersDropdown"

export default {
  title: "components/UserDropdown",
  component: UserDropdown,
  argTypes: {
    onSignOut: { action: "Sign Out" },
  },
}

const Template: Story<UserDropdownProps> = (args: UserDropdownProps) => (
  <Box style={{ backgroundColor: "#000", width: 88 }}>
    <UserDropdown {...args} />
  </Box>
)

export const Closed = Template.bind({})
Closed.args = {
  user: MockUser,
  onSignOut: () => {
    return Promise.resolve()
  },
}

export const Open = Template.bind({})
Open.args = {
  isOpen: true,
  user: MockUser,
  onSignOut: () => {
    return Promise.resolve()
  },
}
