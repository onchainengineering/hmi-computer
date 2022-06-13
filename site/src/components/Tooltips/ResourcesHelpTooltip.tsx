import {
  HelpTooltip,
  HelpTooltipLink,
  HelpTooltipLinksGroup,
  HelpTooltipText,
  HelpTooltipTitle,
} from "./HelpTooltip/HelpTooltip"

export const Language = {
  resourceTooltipTitle: "What is a resource?",
  resourceTooltipText: "A resource is an infrastructure object that is create when the workspace is provisioned.",
  resourceTooltipLink: "Persistent and ephemeral resources",
}

export const ResourcesHelpTooltip: React.FC = () => {
  return (
    <HelpTooltip size="small">
      <HelpTooltipTitle>{Language.resourceTooltipTitle}</HelpTooltipTitle>
      <HelpTooltipText>{Language.resourceTooltipText}</HelpTooltipText>
      <HelpTooltipLinksGroup>
        <HelpTooltipLink href="https://github.com/coder/coder/blob/main/docs/templates.md#persistent-and-ephemeral-resources">
          {Language.resourceTooltipLink}
        </HelpTooltipLink>
      </HelpTooltipLinksGroup>
    </HelpTooltip>
  )
}
