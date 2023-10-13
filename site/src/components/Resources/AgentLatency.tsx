import { type FC, useRef, useState } from "react";
import { useTheme } from "@emotion/react";
import { Theme } from "@mui/material/styles";
import type { WorkspaceAgent, DERPRegion } from "api/typesGenerated";
import {
  HelpTooltipText,
  HelpPopover,
  HelpTooltipTitle,
} from "components/HelpTooltip/HelpTooltip";
import { Stack } from "components/Stack/Stack";
import { getLatencyColor } from "utils/latency";

const getDisplayLatency = (theme: Theme, agent: WorkspaceAgent) => {
  // Find the right latency to display
  const latencyValues = Object.values(agent.latency ?? {});
  const latency =
    latencyValues.find((derp) => derp.preferred) ??
    // Accessing an array index can return undefined as well
    // for some reason TS does not handle that
    (latencyValues[0] as DERPRegion | undefined);

  if (!latency) {
    return undefined;
  }

  return {
    ...latency,
    color: getLatencyColor(theme, latency.latency_ms),
  };
};

export const AgentLatency: FC<{ agent: WorkspaceAgent }> = ({ agent }) => {
  const theme = useTheme();
  const anchorRef = useRef<HTMLButtonElement>(null);
  const [isOpen, setIsOpen] = useState(false);
  const id = isOpen ? "latency-popover" : undefined;
  const latency = getDisplayLatency(theme, agent);

  if (!latency || !agent.latency) {
    return null;
  }

  return (
    <>
      <span
        role="presentation"
        aria-label="latency"
        ref={anchorRef}
        onMouseEnter={() => setIsOpen(true)}
        onMouseLeave={() => setIsOpen(false)}
        css={{ cursor: "pointer", color: latency.color }}
      >
        {Math.round(Math.round(latency.latency_ms))}ms
      </span>
      <HelpPopover
        id={id}
        open={isOpen}
        anchorEl={anchorRef.current}
        onOpen={() => setIsOpen(true)}
        onClose={() => setIsOpen(false)}
      >
        <HelpTooltipTitle>Latency</HelpTooltipTitle>
        <HelpTooltipText>
          This is the latency overhead on non peer to peer connections. The
          first row is the preferred relay.
        </HelpTooltipText>
        <HelpTooltipText>
          <Stack
            direction="column"
            spacing={1}
            css={{ marginTop: theme.spacing(2) }}
          >
            {Object.entries(agent.latency)
              .sort(([, a], [, b]) => a.latency_ms - b.latency_ms)
              .map(([regionName, region]) => (
                <Stack
                  direction="row"
                  key={regionName}
                  spacing={0.5}
                  justifyContent="space-between"
                  css={
                    region.preferred && {
                      color: theme.palette.text.primary,
                    }
                  }
                >
                  <strong>{regionName}</strong>
                  {Math.round(region.latency_ms)}ms
                </Stack>
              ))}
          </Stack>
        </HelpTooltipText>
      </HelpPopover>
    </>
  );
};
