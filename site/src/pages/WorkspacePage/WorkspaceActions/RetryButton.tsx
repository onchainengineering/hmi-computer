import ButtonGroup from "@mui/material/ButtonGroup";
import RetryIcon from "@mui/icons-material/CachedOutlined";
import { type FC } from "react";
import type { Workspace } from "api/typesGenerated";
import { BuildParametersPopover } from "./BuildParametersPopover";
import { TopbarButton } from "components/FullPageLayout/Topbar";
import { ActionButtonProps } from "./Buttons";

type RetryButtonProps = Omit<ActionButtonProps, "loading"> & {
  enableBuildParameters: boolean;
  workspace: Workspace;
};

export const RetryButton: FC<RetryButtonProps> = ({
  handleAction,
  workspace,
  enableBuildParameters,
}) => {
  const mainAction = (
    <TopbarButton startIcon={<RetryIcon />} onClick={() => handleAction()}>
      Retry
    </TopbarButton>
  );

  if (!enableBuildParameters) {
    return mainAction;
  }

  return (
    <ButtonGroup
      variant="outlined"
      css={{
        // Workaround to make the border transitions smoothly on button groups
        "& > button:hover + button": {
          borderLeft: "1px solid #FFF",
        },
      }}
    >
      {mainAction}
      <BuildParametersPopover workspace={workspace} onSubmit={handleAction} />
    </ButtonGroup>
  );
};
