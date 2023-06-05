import Button from "@mui/material/Button"
import { makeStyles } from "@mui/styles"
import WarningIcon from "@mui/icons-material/ErrorOutlineRounded"
import RefreshOutlined from "@mui/icons-material/RefreshOutlined"
import { useMachine } from "@xstate/react"
import { portForwardURL } from "components/PortForwardButton/PortForwardButton"
import { Stack } from "components/Stack/Stack"
import { FC, useCallback, useEffect, useRef, useState } from "react"
import { Helmet } from "react-helmet-async"
import { useNavigate, useParams, useSearchParams } from "react-router-dom"
import { colors } from "theme/colors"
import { v4 as uuidv4 } from "uuid"
import * as XTerm from "xterm"
import { FitAddon } from "xterm-addon-fit"
import { WebLinksAddon } from "xterm-addon-web-links"
import "xterm/css/xterm.css"
import { MONOSPACE_FONT_FAMILY } from "../../theme/constants"
import { pageTitle } from "../../utils/page"
import { terminalMachine } from "../../xServices/terminal/terminalXService"
import { useProxy } from "contexts/ProxyContext"
import { combineClasses } from "utils/combineClasses"
import Box from "@mui/material/Box"

export const Language = {
  workspaceErrorMessagePrefix: "Unable to fetch workspace: ",
  workspaceAgentErrorMessagePrefix: "Unable to fetch workspace agent: ",
  websocketErrorMessagePrefix: "WebSocket failed: ",
}

const TerminalPage: FC = () => {
  const navigate = useNavigate()
  const styles = useStyles()
  const { proxy } = useProxy()
  const { username, workspace: workspaceName } = useParams()
  const xtermRef = useRef<HTMLDivElement>(null)
  const [terminal, setTerminal] = useState<XTerm.Terminal | null>(null)
  const [fitAddon, setFitAddon] = useState<FitAddon | null>(null)
  const [searchParams] = useSearchParams()
  // The reconnection token is a unique token that identifies
  // a terminal session. It's generated by the client to reduce
  // a round-trip, and must be a UUIDv4.
  const reconnectionToken = searchParams.get("reconnect") ?? uuidv4()
  const command = searchParams.get("command") || undefined
  // The workspace name is in the format:
  // <workspace name>[.<agent name>]
  const workspaceNameParts = workspaceName?.split(".")
  const [terminalState, sendEvent] = useMachine(terminalMachine, {
    context: {
      agentName: workspaceNameParts?.[1],
      reconnection: reconnectionToken,
      workspaceName: workspaceNameParts?.[0],
      username: username,
      command: command,
      baseURL: proxy.preferredPathAppURL,
    },
    actions: {
      readMessage: (_, event) => {
        if (typeof event.data === "string") {
          // This exclusively occurs when testing.
          // "jest-websocket-mock" doesn't support ArrayBuffer.
          terminal?.write(event.data)
        } else {
          terminal?.write(new Uint8Array(event.data))
        }
      },
    },
  })
  const isConnected = terminalState.matches("connected")
  const isDisconnected = terminalState.matches("disconnected")
  const {
    workspaceError,
    workspace,
    workspaceAgentError,
    workspaceAgent,
    websocketError,
  } = terminalState.context
  const reloading = useReloading(isDisconnected)
  const shouldDisplayStartupWarning = workspaceAgent
    ? ["starting", "starting_timeout"].includes(workspaceAgent.lifecycle_state)
    : false
  const shouldDisplayStartupError = workspaceAgent
    ? workspaceAgent.lifecycle_state === "start_error"
    : false

  // handleWebLink handles opening of URLs in the terminal!
  const handleWebLink = useCallback(
    (uri: string) => {
      if (
        !workspaceAgent ||
        !workspace ||
        !username ||
        !proxy.preferredWildcardHostname
      ) {
        return
      }

      const open = (uri: string) => {
        // Copied from: https://github.com/xtermjs/xterm.js/blob/master/addons/xterm-addon-web-links/src/WebLinksAddon.ts#L23
        const newWindow = window.open()
        if (newWindow) {
          try {
            newWindow.opener = null
          } catch {
            // no-op, Electron can throw
          }
          newWindow.location.href = uri
        } else {
          console.warn("Opening link blocked as opener could not be cleared")
        }
      }

      try {
        const url = new URL(uri)
        const localHosts = ["0.0.0.0", "127.0.0.1", "localhost"]
        if (!localHosts.includes(url.hostname)) {
          open(uri)
          return
        }
        open(
          portForwardURL(
            proxy.preferredWildcardHostname,
            parseInt(url.port),
            workspaceAgent.name,
            workspace.name,
            username,
          ) + url.pathname,
        )
      } catch (ex) {
        open(uri)
      }
    },
    [workspaceAgent, workspace, username, proxy.preferredWildcardHostname],
  )

  // Create the terminal!
  useEffect(() => {
    if (!xtermRef.current) {
      return
    }
    const terminal = new XTerm.Terminal({
      allowTransparency: true,
      disableStdin: false,
      fontFamily: MONOSPACE_FONT_FAMILY,
      fontSize: 16,
      theme: {
        background: colors.gray[16],
      },
    })
    const fitAddon = new FitAddon()
    setFitAddon(fitAddon)
    terminal.loadAddon(fitAddon)
    terminal.loadAddon(
      new WebLinksAddon((_, uri) => {
        handleWebLink(uri)
      }),
    )
    terminal.onData((data) => {
      sendEvent({
        type: "WRITE",
        request: {
          data: data,
        },
      })
    })
    terminal.onResize((event) => {
      sendEvent({
        type: "WRITE",
        request: {
          height: event.rows,
          width: event.cols,
        },
      })
    })
    setTerminal(terminal)
    terminal.open(xtermRef.current)
    const listener = () => {
      // This will trigger a resize event on the terminal.
      fitAddon.fit()
    }
    window.addEventListener("resize", listener)
    return () => {
      window.removeEventListener("resize", listener)
      terminal.dispose()
    }
  }, [sendEvent, xtermRef, handleWebLink])

  // Triggers the initial terminal connection using
  // the reconnection token and workspace name found
  // from the router.
  useEffect(() => {
    if (searchParams.get("reconnect") === reconnectionToken) {
      return
    }
    searchParams.set("reconnect", reconnectionToken)
    navigate(
      {
        search: searchParams.toString(),
      },
      {
        replace: true,
      },
    )
  }, [searchParams, navigate, reconnectionToken])

  // Apply terminal options based on connection state.
  useEffect(() => {
    if (!terminal || !fitAddon) {
      return
    }

    // We have to fit twice here. It's unknown why, but
    // the first fit will overflow slightly in some
    // scenarios. Applying a second fit resolves this.
    fitAddon.fit()
    fitAddon.fit()

    if (!isConnected) {
      // Disable user input when not connected.
      terminal.options = {
        disableStdin: true,
      }
      if (workspaceError instanceof Error) {
        terminal.writeln(
          Language.workspaceErrorMessagePrefix + workspaceError.message,
        )
      }
      if (workspaceAgentError instanceof Error) {
        terminal.writeln(
          Language.workspaceAgentErrorMessagePrefix +
            workspaceAgentError.message,
        )
      }
      if (websocketError instanceof Error) {
        terminal.writeln(
          Language.websocketErrorMessagePrefix + websocketError.message,
        )
      }
      return
    }

    // The terminal should be cleared on each reconnect
    // because all data is re-rendered from the backend.
    terminal.clear()

    // Focusing on connection allows users to reload the
    // page and start typing immediately.
    terminal.focus()
    terminal.options = {
      disableStdin: false,
      windowsMode: workspaceAgent?.operating_system === "windows",
    }

    // Update the terminal size post-fit.
    sendEvent({
      type: "WRITE",
      request: {
        height: terminal.rows,
        width: terminal.cols,
      },
    })
  }, [
    workspaceError,
    workspaceAgentError,
    websocketError,
    workspaceAgent,
    terminal,
    fitAddon,
    isConnected,
    sendEvent,
  ])

  return (
    <>
      <Helmet>
        <title>
          {terminalState.context.workspace
            ? pageTitle(
                `Terminal · ${terminalState.context.workspace.owner_name}/${terminalState.context.workspace.name}`,
              )
            : ""}
        </title>
      </Helmet>
      {/* This overlay makes it more obvious that the terminal is disconnected. */}
      {/* It's nice for situations where Coder restarts, and they are temporarily disconnected. */}
      <div className={`${styles.overlay} ${isDisconnected ? "" : "connected"}`}>
        {reloading.status === "reloading" ? (
          <span className={styles.overlayText}>Reloading...</span>
        ) : (
          <Stack spacing={0.5} alignItems="center">
            <span className={styles.overlayText}>Disconnected</span>
            <span className={styles.overlaySubtext}>
              Press any key to retry
            </span>
          </Stack>
        )}
      </div>
      {shouldDisplayStartupError && (
        <div
          className={combineClasses([styles.alert, styles.alertError])}
          role="alert"
        >
          <WarningIcon className={styles.alertIcon} />
          <div>
            <div className={styles.alertTitle}>Startup script failed</div>
            <div className={styles.alertMessage}>
              You can continue using this terminal, but something may be missing
              or not fully set up.
            </div>
          </div>
        </div>
      )}
      <Box display="flex" flexDirection="column" height="100vh">
        {shouldDisplayStartupWarning && (
          <div className={styles.alert} role="alert">
            <WarningIcon className={styles.alertIcon} />
            <div>
              <div className={styles.alertTitle}>
                Startup script is still running
              </div>
              <div className={styles.alertMessage}>
                You can continue using this terminal, but something may be
                missing or not fully set up.
              </div>
            </div>
            <div className={styles.alertActions}>
              <Button
                startIcon={<RefreshOutlined />}
                size="small"
                onClick={() => {
                  // By redirecting the user without the session in the URL we
                  // create a new one
                  window.location.href = window.location.pathname
                }}
              >
                Refresh session
              </Button>
            </div>
          </div>
        )}
        <div
          className={styles.terminal}
          ref={xtermRef}
          data-testid="terminal"
        />
      </Box>
    </>
  )
}

const useReloading = (isDisconnected: boolean) => {
  const [status, setStatus] = useState<"reloading" | "notReloading">(
    "notReloading",
  )

  // Retry connection on key press when it is disconnected
  useEffect(() => {
    if (!isDisconnected) {
      return
    }

    const keyDownHandler = () => {
      setStatus("reloading")
      window.location.reload()
    }

    document.addEventListener("keydown", keyDownHandler)

    return () => {
      document.removeEventListener("keydown", keyDownHandler)
    }
  }, [isDisconnected])

  return {
    status,
  }
}

const useStyles = makeStyles((theme) => ({
  overlay: {
    position: "absolute",
    pointerEvents: "none",
    top: 0,
    left: 0,
    bottom: 0,
    right: 0,
    zIndex: 1,
    alignItems: "center",
    justifyContent: "center",
    display: "flex",
    color: "white",
    fontSize: 16,
    backgroundColor: "rgba(0, 0, 0, 0.6)",
    backdropFilter: "blur(4px)",
    "&.connected": {
      opacity: 0,
    },
  },
  overlayText: {
    fontSize: 16,
    fontWeight: 600,
  },
  overlaySubtext: {
    fontSize: 14,
    color: theme.palette.text.secondary,
  },
  terminal: {
    width: "100vw",
    overflow: "hidden",
    padding: theme.spacing(1),
    backgroundColor: theme.palette.background.paper,
    flex: 1,
    // These styles attempt to mimic the VS Code scrollbar.
    "& .xterm": {
      padding: 4,
      width: "100vw",
      height: "100vh",
    },
    "& .xterm-viewport": {
      // This is required to force full-width on the terminal.
      // Otherwise there's a small white bar to the right of the scrollbar.
      width: "auto !important",
    },
    "& .xterm-viewport::-webkit-scrollbar": {
      width: "10px",
    },
    "& .xterm-viewport::-webkit-scrollbar-track": {
      backgroundColor: "inherit",
    },
    "& .xterm-viewport::-webkit-scrollbar-thumb": {
      minHeight: 20,
      backgroundColor: "rgba(255, 255, 255, 0.18)",
    },
  },
  alert: {
    display: "flex",
    background: theme.palette.background.paperLight,
    alignItems: "center",
    padding: theme.spacing(2),
    gap: theme.spacing(2),
    borderBottom: `1px solid ${theme.palette.divider}`,
    ...theme.typography.body2,
  },
  alertIcon: {
    color: theme.palette.warning.light,
    fontSize: theme.spacing(3),
  },
  alertError: {
    "& $alertIcon": {
      color: theme.palette.error.light,
    },
  },
  alertTitle: {
    fontWeight: 600,
    color: theme.palette.text.primary,
  },
  alertMessage: {
    fontSize: 14,
    color: theme.palette.text.secondary,
  },
  alertActions: {
    marginLeft: "auto",
  },
}))

export default TerminalPage
