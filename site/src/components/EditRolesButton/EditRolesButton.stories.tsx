import { ComponentMeta, Story } from "@storybook/react"
import {
  MockOwnerRole,
  MockSiteRoles,
  MockUserAdminRole,
} from "testHelpers/entities"
import { EditRolesButtonProps, EditRolesButton } from "./EditRolesButton"

export default {
  title: "components/EditRolesButton",
  component: EditRolesButton,
  argTypes: {
    defaultIsOpen: {
      defaultValue: true,
    },
  },
} as ComponentMeta<typeof EditRolesButton>

const Template: Story<EditRolesButtonProps> = (args) => (
  <EditRolesButton {...args} />
)

export const Open = Template.bind({})
Open.args = {
  roles: MockSiteRoles,
  selectedRoles: [MockUserAdminRole, MockOwnerRole],
  defaultIsOpen: true,
}
Open.play = async () => {
  //👇 This sets a timeout of 2s
  await new Promise((resolve) => setTimeout(resolve, 2000));
};

export const Loading = Template.bind({})
Loading.args = {
  isLoading: true,
  roles: MockSiteRoles,
  selectedRoles: [MockUserAdminRole, MockOwnerRole],
}
Loading.parameters = {
  chromatic: { delay: 300 },
}
