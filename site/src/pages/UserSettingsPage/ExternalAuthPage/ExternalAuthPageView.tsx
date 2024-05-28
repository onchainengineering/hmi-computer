import { useTheme } from "@emotion/react";
import AutorenewIcon from "@mui/icons-material/Autorenew";
import LoadingButton from "@mui/lab/LoadingButton";
import Badge from "@mui/material/Badge";
import Divider from "@mui/material/Divider";
import { styled } from "@mui/material/styles";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import Tooltip from "@mui/material/Tooltip";
import visuallyHidden from "@mui/utils/visuallyHidden";
import { type FC, useState, useCallback, useEffect } from "react";
import { useQuery } from "react-query";
import { externalAuthProvider } from "api/queries/externalAuth";
import type {
  ListUserExternalAuthResponse,
  ExternalAuthLinkProvider,
  ExternalAuthLink,
} from "api/typesGenerated";
import { ErrorAlert } from "components/Alert/ErrorAlert";
import { Avatar } from "components/Avatar/Avatar";
import { AvatarData } from "components/AvatarData/AvatarData";
import { Loader } from "components/Loader/Loader";
import {
  MoreMenu,
  MoreMenuContent,
  MoreMenuItem,
  MoreMenuTrigger,
  ThreeDotsButton,
} from "components/MoreMenu/MoreMenu";
import { TableEmpty } from "components/TableEmpty/TableEmpty";
import type { ExternalAuthPollingState } from "pages/CreateWorkspacePage/CreateWorkspacePage";

export type ExternalAuthPageViewProps = {
  isLoading: boolean;
  getAuthsError?: unknown;
  unlinked: number;
  auths?: ListUserExternalAuthResponse;
  onUnlinkExternalAuth: (provider: string) => void;
  onValidateExternalAuth: (provider: string) => void;
};

export const ExternalAuthPageView: FC<ExternalAuthPageViewProps> = ({
  isLoading,
  getAuthsError,
  auths,
  unlinked,
  onUnlinkExternalAuth,
  onValidateExternalAuth,
}) => {
  if (getAuthsError) {
    // Nothing to show if there is an error
    return <ErrorAlert error={getAuthsError} />;
  }

  if (isLoading || !auths) {
    return <Loader fullscreen />;
  }

  return (
    <>
      <TableContainer>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Application</TableCell>
              <TableCell>
                <span aria-hidden css={{ ...visuallyHidden }}>
                  Link to connect
                </span>
              </TableCell>
              <TableCell width="1%" />
            </TableRow>
          </TableHead>
          <TableBody>
            {auths.providers === null || auths.providers?.length === 0 ? (
              <TableEmpty message="No providers have been configured" />
            ) : (
              auths.providers?.map((app) => (
                <ExternalAuthRow
                  key={app.id}
                  app={app}
                  unlinked={unlinked}
                  link={auths.links.find((l) => l.provider_id === app.id)}
                  onUnlinkExternalAuth={() => {
                    onUnlinkExternalAuth(app.id);
                  }}
                  onValidateExternalAuth={() => {
                    onValidateExternalAuth(app.id);
                  }}
                />
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </>
  );
};

interface ExternalAuthRowProps {
  app: ExternalAuthLinkProvider;
  link?: ExternalAuthLink;
  unlinked: number;
  onUnlinkExternalAuth: () => void;
  onValidateExternalAuth: () => void;
}

const StyledBadge = styled(Badge)(({ theme }) => ({
  "& .MuiBadge-badge": {
    // Make a circular background for the icon. Background provides contrast, with a thin
    // border to separate it from the avatar image.
    backgroundColor: `${theme.palette.background.paper}`,
    borderStyle: "solid",
    borderColor: `${theme.palette.secondary.main}`,
    borderWidth: "thin",

    // Override the default minimum sizes, as they are larger than what we want.
    minHeight: "0px",
    minWidth: "0px",
    // Override the default "height", which is usually set to some constant value.
    height: "auto",
    // Padding adds some room for the icon to live in.
    padding: "0.1em",
  },
}));

const ExternalAuthRow: FC<ExternalAuthRowProps> = ({
  app,
  unlinked,
  link,
  onUnlinkExternalAuth,
  onValidateExternalAuth,
}) => {
  const theme = useTheme();
  const name = app.display_name || app.id || app.type;
  const authURL = "/external-auth/" + app.id;

  const {
    externalAuth,
    externalAuthPollingState,
    refetch,
    startPollingExternalAuth,
  } = useExternalAuth(app.id, unlinked);

  const authenticated = externalAuth
    ? externalAuth.authenticated
    : link?.authenticated ?? false;

  let avatar = app.display_icon ? (
    <Avatar src={app.display_icon} variant="square" fitImage size="sm" />
  ) : (
    <Avatar>{name}</Avatar>
  );

  // If the link is authenticated and has a refresh token, show that it will automatically
  // attempt to authenticate when the token expires.
  if (link?.has_refresh_token && authenticated) {
    avatar = (
      <StyledBadge
        anchorOrigin={{
          vertical: "bottom",
          horizontal: "right",
        }}
        color="default"
        overlap="circular"
        badgeContent={
          <Tooltip
            title="Authentication token will automatically refresh when expired."
            placement="right"
          >
            <AutorenewIcon
              sx={{
                fontSize: "1em",
              }}
            />
          </Tooltip>
        }
      >
        {avatar}
      </StyledBadge>
    );
  }

  return (
    <TableRow key={app.id}>
      <TableCell>
        <AvatarData title={name} avatar={avatar} />
        {link?.validate_error && (
          <>
            <span
              css={{ paddingLeft: "1em", color: theme.palette.error.light }}
            >
              Error:{" "}
            </span>
            {link?.validate_error}
          </>
        )}
      </TableCell>
      <TableCell css={{ textAlign: "right" }}>
        <LoadingButton
          disabled={authenticated}
          variant="contained"
          loading={externalAuthPollingState === "polling"}
          onClick={() => {
            window.open(authURL, "_blank", "width=900,height=600");
            startPollingExternalAuth();
          }}
        >
          {authenticated ? "Authenticated" : "Click to Login"}
        </LoadingButton>
      </TableCell>
      <TableCell>
        <MoreMenu>
          <MoreMenuTrigger>
            <ThreeDotsButton size="small" disabled={!authenticated} />
          </MoreMenuTrigger>
          <MoreMenuContent>
            <MoreMenuItem
              onClick={async () => {
                onValidateExternalAuth();
                // This is kinda jank. It does a refetch of the thing
                // it just validated... But we need to refetch to update the
                // login button. And the 'onValidateExternalAuth' does the
                // message display.
                await refetch();
              }}
            >
              Test Validate&hellip;
            </MoreMenuItem>
            <Divider />
            <MoreMenuItem
              danger
              onClick={async () => {
                onUnlinkExternalAuth();
                await refetch();
              }}
            >
              Unlink&hellip;
            </MoreMenuItem>
          </MoreMenuContent>
        </MoreMenu>
      </TableCell>
    </TableRow>
  );
};

// useExternalAuth handles the polling of the auth to update the button.
const useExternalAuth = (providerID: string, unlinked: number) => {
  const [externalAuthPollingState, setExternalAuthPollingState] =
    useState<ExternalAuthPollingState>("idle");

  const startPollingExternalAuth = useCallback(() => {
    setExternalAuthPollingState("polling");
  }, []);

  const { data: externalAuth, refetch } = useQuery({
    ...externalAuthProvider(providerID),
    refetchInterval: externalAuthPollingState === "polling" ? 1000 : false,
  });

  const signedIn = externalAuth?.authenticated;

  useEffect(() => {
    if (unlinked > 0) {
      void refetch();
    }
  }, [refetch, unlinked]);

  useEffect(() => {
    if (signedIn) {
      setExternalAuthPollingState("idle");
      return;
    }

    if (externalAuthPollingState !== "polling") {
      return;
    }

    // Poll for a maximum of one minute
    const quitPolling = setTimeout(
      () => setExternalAuthPollingState("abandoned"),
      60_000,
    );
    return () => {
      clearTimeout(quitPolling);
    };
  }, [externalAuthPollingState, signedIn]);

  return {
    startPollingExternalAuth,
    externalAuth,
    externalAuthPollingState,
    refetch,
  };
};
