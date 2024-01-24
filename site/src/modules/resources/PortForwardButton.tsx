import Button from "@mui/material/Button";
import Link from "@mui/material/Link";
import CircularProgress from "@mui/material/CircularProgress";
import OpenInNewOutlined from "@mui/icons-material/OpenInNewOutlined";
import { type Interpolation, type Theme, useTheme } from "@emotion/react";
import type { FC } from "react";
import { useQuery } from "react-query";
import { docs } from "utils/docs";
import { getAgentListeningPorts } from "api/api";
import type {
  WorkspaceAgent,
  WorkspaceAgentListeningPort,
  WorkspaceAgentListeningPortsResponse,
} from "api/typesGenerated";
import { portForwardURL } from "utils/portForward";
import { type ClassName, useClassName } from "hooks/useClassName";
import {
  HelpTooltipLink,
  HelpTooltipLinksGroup,
  HelpTooltipText,
  HelpTooltipTitle,
} from "components/HelpTooltip/HelpTooltip";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "components/Popover/Popover";
import KeyboardArrowDown from "@mui/icons-material/KeyboardArrowDown";
import Stack from "@mui/material/Stack";
import Select from "@mui/material/Select";
import MenuItem from "@mui/material/MenuItem";
import FormControl from "@mui/material/FormControl";
import TextField from "@mui/material/TextField";
import SensorsIcon from '@mui/icons-material/Sensors';
import Add from '@mui/icons-material/Add';
import IconButton from "@mui/material/IconButton";
import LockIcon from '@mui/icons-material/Lock';
import LockOpenIcon from '@mui/icons-material/LockOpen';
import DeleteIcon from '@mui/icons-material/Delete';

export interface PortForwardButtonProps {
  host: string;
  username: string;
  workspaceName: string;
  agent: WorkspaceAgent;

  /**
   * Only for use in Storybook
   */
  storybook?: {
    portsQueryData?: WorkspaceAgentListeningPortsResponse;
  };
}

export const PortForwardButton: FC<PortForwardButtonProps> = (props) => {
  const { agent, storybook } = props;

  const paper = useClassName(classNames.paper, []);

  const portsQuery = useQuery({
    queryKey: ["portForward", agent.id],
    queryFn: () => getAgentListeningPorts(agent.id),
    enabled: !storybook && agent.status === "connected",
    refetchInterval: 5_000,
  });

  const data = storybook ? storybook.portsQueryData : portsQuery.data;

  return (
    <Popover>
      <PopoverTrigger>
        <Button
          disabled={!data}
          size="small"
          variant="text"
          endIcon={<KeyboardArrowDown />}
          css={{ fontSize: 13, padding: "8px 12px" }}
          startIcon={
            data ? (
              <div>
                <span css={styles.portCount}>{data.ports.length}</span>
              </div>
            ) : (
              <CircularProgress size={10} />
            )
          }
        >
          Open ports
        </Button>
      </PopoverTrigger>
      <PopoverContent horizontal="right" classes={{ paper }}>
        <PortForwardPopoverView {...props} ports={data?.ports} />
      </PopoverContent>
    </Popover>
  );
};

interface PortForwardPopoverViewProps extends PortForwardButtonProps {
  ports?: WorkspaceAgentListeningPort[];
}

export const PortForwardPopoverView: FC<PortForwardPopoverViewProps> = ({
  host,
  workspaceName,
  agent,
  username,
  ports,
}) => {
  const theme = useTheme();
  const sharedPorts = [
    {
      port: 8090,
      share_level: "Authenticated",
    },
    {
      port: 8091,
      share_level: "Public",
    }
  ];

  return (
    <>
      <div
        css={{
          padding: 20,
          borderBottom: `1px solid ${theme.palette.divider}`,
        }}
      >
        <Stack direction="row" justifyContent="space-between" alignItems="start">
        <HelpTooltipTitle>Listening ports</HelpTooltipTitle>
          <HelpTooltipLink href={docs("/networking/port-forwarding#dashboard")}>
            Learn more
          </HelpTooltipLink>
        </Stack>
        <HelpTooltipText css={{ color: theme.palette.text.secondary }}>
          {ports?.length === 0
            ? "No open ports were detected."
            : "The listening ports are exclusively accessible to you."
          }

        </HelpTooltipText>
        <form
          css={styles.newPortForm}
          onSubmit={(e) => {
            e.preventDefault();
            const formData = new FormData(e.currentTarget);
            const port = Number(formData.get("portNumber"));
            const url = portForwardURL(
              host,
              port,
              agent.name,
              workspaceName,
              username,
            );
            window.open(url, "_blank");
          }}
        >
          <input
            aria-label="Port number"
            name="portNumber"
            type="number"
            placeholder="Connect to port..."
            min={0}
            max={65535}
            required
            css={styles.newPortInput}
          />
          <Button
            type="submit"
            size="small"
            variant="text"
            css={{
              paddingLeft: 12,
              paddingRight: 12,
              minWidth: 0,
            }}
          >
            <OpenInNewOutlined
              css={{
                flexShrink: 0,
                width: 14,
                height: 14,
                color: theme.palette.text.primary,
              }}
            />
          </Button>
        </form>
        <div
        css={{
          paddingTop: 10,
        }}>
          {ports?.map((port) => {
            const url = portForwardURL(
              host,
              port.port,
              agent.name,
              workspaceName,
              username,
            );
            const label =
              port.process_name !== "" ? port.process_name : port.port;
            return (
              <Stack key={port.port} direction="row" justifyContent="space-between" alignItems="center">
                <Link
                  underline="none"
                  css={styles.portLink}
                  href={url}
                  target="_blank"
                  rel="noreferrer"
                >
                  <SensorsIcon css={{ width: 14, height: 14 }} />
                  {label}
                </Link>
                <Link
                  underline="none"
                  css={styles.portLink}
                  href={url}
                  target="_blank"
                  rel="noreferrer"
                >
                  <span css={styles.portNumber}>{port.port}</span>
                </Link>
                <Button size="small" variant="text">
                  Share
                </Button>
              </Stack>
            );
          })}
        </div>
        </div>
        <div css={{
          padding: 20,
        }}>
        <HelpTooltipTitle>Shared Ports</HelpTooltipTitle>
        <HelpTooltipText css={{ color: theme.palette.text.secondary }}>
          {ports?.length === 0
            ? "No ports are shared."
            : "Ports can be shared with other Coder users or with the public."}
        </HelpTooltipText>
        <div>
          {sharedPorts?.map((port) => {
            const url = portForwardURL(
              host,
              port.port,
              agent.name,
              workspaceName,
              username,
            );
            const label = port.port;
            return (
              <Stack key={port.port} direction="row" justifyContent="space-between" alignItems="center">
                <Link
                  underline="none"
                  css={styles.portLink}
                  href={url}
                  target="_blank"
                  rel="noreferrer"
                >
                  {port.share_level === "Public" ?
                  (
                    <LockOpenIcon css={{ width: 14, height: 14 }} />
                  )
                  : (
                    <LockIcon css={{ width: 14, height: 14 }} />
                  )}
                  {label}
                </Link>
                <Stack direction="row" gap={1}>
                <FormControl size="small">
                  <Select
                    sx={{
                      boxShadow: "none",
                      ".MuiOutlinedInput-notchedOutline": { border: 0 },
                      "&.MuiOutlinedInput-root:hover .MuiOutlinedInput-notchedOutline":
                        {
                          border: 0,
                        },
                      "&.MuiOutlinedInput-root.Mui-focused .MuiOutlinedInput-notchedOutline":
                        {
                          border: 0,
                        },
                    }}
                    value={port.share_level}
                  >
                    <MenuItem value="Owner">Owner</MenuItem>
                    <MenuItem value="Authenticated">Authenticated</MenuItem>
                    <MenuItem value="Public">Public</MenuItem>
                  </Select>
                </FormControl>
                </Stack>

              </Stack>
            );
          })}
          </div>
          <Stack direction="column" gap={1} justifyContent="flex-end" sx={{
          marginTop: 2,
        }}>
          <TextField
            label="Port"
            variant="outlined"
            size="small"
          />
          <FormControl size="small">
                  <Select
                    value="Authenticated"
                  >
                    <MenuItem value="Authenticated">Authenticated</MenuItem>
                    <MenuItem value="Public">Public</MenuItem>
                  </Select>
          </FormControl>
          <Button variant="contained">
            Add Shared Port
          </Button>
        </Stack>
      </div>


      {/* <div css={{ padding: 20 }}>
        <HelpTooltipTitle>Forward port</HelpTooltipTitle>
        <HelpTooltipText css={{ color: theme.palette.text.secondary }}>
          Access ports running on the agent:
        </HelpTooltipText>

        <form
          css={styles.newPortForm}
          onSubmit={(e) => {
            e.preventDefault();
            const formData = new FormData(e.currentTarget);
            const port = Number(formData.get("portNumber"));
            const url = portForwardURL(
              host,
              port,
              agent.name,
              workspaceName,
              username,
            );
            window.open(url, "_blank");
          }}
        >
          <input
            aria-label="Port number"
            name="portNumber"
            type="number"
            placeholder="Type a port number..."
            min={0}
            max={65535}
            required
            css={styles.newPortInput}
          />
          <Button
            type="submit"
            size="small"
            variant="text"
            css={{
              paddingLeft: 12,
              paddingRight: 12,
              minWidth: 0,
            }}
          >
            <OpenInNewOutlined
              css={{
                flexShrink: 0,
                width: 14,
                height: 14,
                color: theme.palette.text.primary,
              }}
            />
          </Button>
        </form>

        <HelpTooltipLinksGroup>
          <HelpTooltipLink href={docs("/networking/port-forwarding#dashboard")}>
            Learn more
          </HelpTooltipLink>
        </HelpTooltipLinksGroup>
      </div> */}
    </>
  );
};

const classNames = {
  paper: (css, theme) => css`
    padding: 0;
    width: 304px;
    color: ${theme.palette.text.secondary};
    margin-top: 4px;
  `,
} satisfies Record<string, ClassName>;

const styles = {
  portCount: (theme) => ({
    fontSize: 12,
    fontWeight: 500,
    height: 20,
    minWidth: 20,
    padding: "0 4px",
    borderRadius: "50%",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: theme.palette.action.selected,
  }),

  portLink: (theme) => ({
    color: theme.palette.text.primary,
    fontSize: 14,
    display: "flex",
    alignItems: "center",
    gap: 8,
    paddingTop: 8,
    paddingBottom: 8,
    fontWeight: 500,
  }),

  portNumber: (theme) => ({
    marginLeft: "auto",
    color: theme.palette.text.secondary,
    fontSize: 13,
    fontWeight: 400,
  }),

  newPortForm: (theme) => ({
    border: `1px solid ${theme.palette.divider}`,
    borderRadius: "4px",
    marginTop: 8,
    display: "flex",
    alignItems: "center",
    "&:focus-within": {
      borderColor: theme.palette.primary.main,
    },
  }),

  newPortInput: (theme) => ({
    fontSize: 14,
    height: 34,
    padding: "0 12px",
    background: "none",
    border: 0,
    outline: "none",
    color: theme.palette.text.primary,
    appearance: "textfield",
    display: "block",
    width: "100%",
  }),
} satisfies Record<string, Interpolation<Theme>>;
