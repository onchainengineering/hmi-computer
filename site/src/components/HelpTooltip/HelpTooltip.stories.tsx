import type { Meta, StoryObj } from "@storybook/react";
import {
	HelpTooltip,
	HelpTooltipLink,
	HelpTooltipLinksGroup,
	HelpTooltipText,
	HelpTooltipTitle,
} from "./HelpTooltip";

const meta: Meta<typeof HelpTooltip> = {
	title: "components/HelpTooltip",
	component: HelpTooltip,
	args: {
		children: (
			<>
				<HelpTooltipTitle>What is a template?</HelpTooltipTitle>
				<HelpTooltipText>
					A template is a common configuration for your team&apos;s workspaces.
				</HelpTooltipText>
				<HelpTooltipLinksGroup>
					<HelpTooltipLink href="https://github.com/onchainengineering/hmi-computer/">
						Creating a template
					</HelpTooltipLink>
					<HelpTooltipLink href="https://github.com/onchainengineering/hmi-computer/">
						Updating a template
					</HelpTooltipLink>
				</HelpTooltipLinksGroup>
			</>
		),
	},
};

export default meta;
type Story = StoryObj<typeof HelpTooltip>;

const Example: Story = {};

export { Example as HelpTooltip };
