import { DeploymentStats, WorkspaceStatus } from "api/typesGenerated"
import { FC, useMemo, useEffect, useState } from "react"
import prettyBytes from "pretty-bytes"
import { getStatus } from "components/WorkspaceStatusBadge/WorkspaceStatusBadge"
import BuildingIcon from "@material-ui/icons/Build"
import { makeStyles } from "@material-ui/core/styles"
import { RocketIcon } from "components/Icons/RocketIcon"
import { MONOSPACE_FONT_FAMILY } from "theme/constants"
import Tooltip from "@material-ui/core/Tooltip"
import { Link as RouterLink } from "react-router-dom"
import Link from "@material-ui/core/Link"
import InfoIcon from "@material-ui/icons/InfoOutlined"
import { VSCodeIcon } from "components/Icons/VSCodeIcon"
import DownloadIcon from "@material-ui/icons/CloudDownload"
import UploadIcon from "@material-ui/icons/CloudUpload"
import LatencyIcon from "@material-ui/icons/SettingsEthernet"
import WebTerminalIcon from "@material-ui/icons/WebAsset"
import { TerminalIcon } from "components/Icons/TerminalIcon"
import dayjs from "dayjs"
import CollectedIcon from "@material-ui/icons/Compare"
import RefreshIcon from "@material-ui/icons/Refresh"

export interface DeploymentBannerViewProps {
  fetchStats?: () => void
  stats?: DeploymentStats
}

export const DeploymentBannerView: FC<DeploymentBannerViewProps> = ({
  stats,
  fetchStats,
}) => {
  const styles = useStyles()
  const aggregatedMinutes = useMemo(() => {
    if (!stats) {
      return
    }
    return dayjs(stats.collected_at).diff(stats.aggregated_since, "minutes")
  }, [stats])
  const displayLatency = stats?.workspace_connection_latency_ms.P50 || -1
  const [timeUntilRefresh, setTimeUntilRefresh] = useState(0)
  useEffect(() => {
    if (!stats || !fetchStats) {
      return
    }

    let timeUntilRefresh = dayjs(stats.refreshing_at).diff(
      stats.collected_at,
      "seconds",
    )
    setTimeUntilRefresh(timeUntilRefresh)
    let canceled = false
    const loop = () => {
      if (canceled) {
        return
      }
      setTimeUntilRefresh(timeUntilRefresh--)
      if (timeUntilRefresh > 0) {
        return setTimeout(loop, 1000)
      }
      fetchStats()
    }
    const timeout = setTimeout(loop, 1000)
    return () => {
      canceled = true
      clearTimeout(timeout)
    }
  }, [fetchStats, stats])
  const lastAggregated = useMemo(() => {
    if (!stats) {
      return
    }
    if (!fetchStats) {
      // Storybook!
      return "just now"
    }
    return dayjs().to(dayjs(stats.collected_at))
    // eslint-disable-next-line react-hooks/exhaustive-deps -- We want this to periodically update!
  }, [timeUntilRefresh, stats])

  return (
    <div className={styles.container}>
      <Tooltip title="Status of your Coder deployment. Only visible for admins!">
        <div className={styles.rocket}>
          <RocketIcon />
        </div>
      </Tooltip>
      <div className={styles.group}>
        <div className={styles.category}>Workspaces</div>
        <div className={styles.values}>
          <WorkspaceBuildValue
            status="pending"
            count={stats?.pending_workspaces}
          />
          <ValueSeparator />
          <WorkspaceBuildValue
            status="starting"
            count={stats?.building_workspaces}
          />
          <ValueSeparator />
          <WorkspaceBuildValue
            status="running"
            count={stats?.running_workspaces}
          />
          <ValueSeparator />
          <WorkspaceBuildValue
            status="stopped"
            count={stats?.stopped_workspaces}
          />
          <ValueSeparator />
          <WorkspaceBuildValue
            status="failed"
            count={stats?.failed_workspaces}
          />
        </div>
      </div>
      <div className={styles.group}>
        <Tooltip title={`Activity in the last ~${aggregatedMinutes} minutes`}>
          <div className={styles.category}>
            Transmission
            <InfoIcon />
          </div>
        </Tooltip>

        <div className={styles.values}>
          <Tooltip title="Data sent through workspace workspaces">
            <div className={styles.value}>
              <DownloadIcon />
              {stats ? prettyBytes(stats.workspace_rx_bytes) : "-"}
            </div>
          </Tooltip>
          <ValueSeparator />
          <Tooltip title="Data sent from workspace connections">
            <div className={styles.value}>
              <UploadIcon />
              {stats ? prettyBytes(stats.workspace_tx_bytes) : "-"}
            </div>
          </Tooltip>
          <ValueSeparator />
          <Tooltip
            title={
              displayLatency < 0
                ? "No recent workspace connections have been made"
                : "The average latency of user connections to workspaces"
            }
          >
            <div className={styles.value}>
              <LatencyIcon />
              {displayLatency > 0 ? displayLatency?.toFixed(2) + " ms" : "-"}
            </div>
          </Tooltip>
        </div>
      </div>
      <div className={styles.group}>
        <div className={styles.category}>Active Connections</div>

        <div className={styles.values}>
          <Tooltip title="VS Code Editors with the Coder Remote Extension">
            <div className={styles.value}>
              <VSCodeIcon className={styles.iconStripColor} />
              {typeof stats?.session_count_vscode === "undefined"
                ? "-"
                : stats?.session_count_vscode}
            </div>
          </Tooltip>
          <ValueSeparator />
          <Tooltip title="SSH Sessions">
            <div className={styles.value}>
              <TerminalIcon />
              {typeof stats?.session_count_ssh === "undefined"
                ? "-"
                : stats?.session_count_ssh}
            </div>
          </Tooltip>
          <ValueSeparator />
          <Tooltip title="Web Terminal Sessions">
            <div className={styles.value}>
              <WebTerminalIcon />
              {typeof stats?.session_count_reconnecting_pty === "undefined"
                ? "-"
                : stats?.session_count_reconnecting_pty}
            </div>
          </Tooltip>
        </div>
      </div>
      <div className={styles.refresh}>
        <Tooltip title="The last time stats were aggregated. Workspaces report statistics periodically, so it may take a bit for these to update!">
          <div className={styles.value}>
            <CollectedIcon />
            {lastAggregated}
          </div>
        </Tooltip>

        <Tooltip title="A countdown until stats are refetched">
          <div className={styles.value}>
            <RefreshIcon />
            {timeUntilRefresh}s
          </div>
        </Tooltip>
      </div>
    </div>
  )
}

const ValueSeparator: FC = () => {
  const styles = useStyles()
  return <div className={styles.valueSeparator}>/</div>
}

const WorkspaceBuildValue: FC<{
  status: WorkspaceStatus
  count?: number
}> = ({ status, count }) => {
  const styles = useStyles()
  const displayStatus = getStatus(status)
  let statusText = displayStatus.text
  let icon = displayStatus.icon
  if (status === "starting") {
    icon = <BuildingIcon />
    statusText = "Building"
  }

  return (
    <Tooltip title={`${statusText} Workspaces`}>
      <Link
        component={RouterLink}
        to={`/workspaces?filter=${encodeURIComponent("status:" + status)}`}
      >
        <div className={styles.value}>
          {icon}
          {typeof count === "undefined" ? "-" : count}
        </div>
      </Link>
    </Tooltip>
  )
}

const useStyles = makeStyles((theme) => ({
  rocket: {
    display: "flex",
    alignItems: "center",

    "& svg": {
      width: 16,
      height: 16,
    },
  },
  container: {
    position: "sticky",
    bottom: 0,
    zIndex: 1,
    padding: theme.spacing(1, 2),
    backgroundColor: theme.palette.background.paper,
    display: "flex",
    alignItems: "center",
    fontFamily: MONOSPACE_FONT_FAMILY,
    fontSize: 12,
    gap: theme.spacing(4),
    borderTop: `1px solid ${theme.palette.divider}`,
  },
  group: {
    display: "flex",
    alignItems: "center",
  },
  category: {
    marginRight: theme.spacing(2),
    color: theme.palette.text.hint,

    "& svg": {
      width: 12,
      height: 12,
      marginBottom: 2,
    },
  },
  values: {
    display: "flex",
    gap: theme.spacing(1),
    color: theme.palette.text.secondary,
  },
  valueSeparator: {
    color: theme.palette.text.disabled,
  },
  value: {
    display: "flex",
    alignItems: "center",
    gap: theme.spacing(0.5),

    "& svg": {
      width: 12,
      height: 12,
    },
  },
  iconStripColor: {
    "& *": {
      fill: "currentColor",
    },
  },
  refresh: {
    color: theme.palette.text.hint,
    marginLeft: "auto",
    display: "flex",
    alignItems: "center",
    gap: theme.spacing(2),
  },
}))
