import Collapse from "@material-ui/core/Collapse"
import { makeStyles } from "@material-ui/core/styles"
import TableCell from "@material-ui/core/TableCell"
import { AuditLog } from "api/typesGenerated"
import {
  CloseDropdown,
  OpenDropdown,
} from "components/DropdownArrows/DropdownArrows"
import { Pill } from "components/Pill/Pill"
import { Stack } from "components/Stack/Stack"
import { TimelineEntry } from "components/Timeline/TimelineEntry"
import { UserAvatar } from "components/UserAvatar/UserAvatar"
import { useState } from "react"
import { PaletteIndex } from "theme/palettes"
import userAgentParser from "ua-parser-js"
import { AuditLogDiff } from "./AuditLogDiff"
import i18next from "i18next"
import { AuditLogDescription } from "./AuditLogDescription"

const httpStatusColor = (httpStatus: number): PaletteIndex => {
  if (httpStatus >= 300 && httpStatus < 500) {
    return "warning"
  }

  if (httpStatus >= 500) {
    return "error"
  }

  return "success"
}

export interface AuditLogRowProps {
  auditLog: AuditLog
  // Useful for Storybook
  defaultIsDiffOpen?: boolean
}

interface GroupMember {
  user_id: string
  group_id: string
}

export const AuditLogRow: React.FC<AuditLogRowProps> = ({
  auditLog,
  defaultIsDiffOpen = false,
}) => {
  const styles = useStyles()
  const { t } = i18next
  const [isDiffOpen, setIsDiffOpen] = useState(defaultIsDiffOpen)
  const diffs = Object.entries(auditLog.diff)
  const shouldDisplayDiff = diffs.length > 0
  const { os, browser } = userAgentParser(auditLog.user_agent)
  const displayBrowserInfo = browser.name
    ? `${browser.name} ${browser.version}`
    : t("auditLog:table.logRow.notAvailable")

  let auditDiff = auditLog.diff
  // groups have nested diffs (group members)
  // so we overwrite the member diff such that
  // only the user_id is shown.
  if (auditLog.resource_type === "group") {
    auditDiff = {
      ...auditLog.diff,
      members: {
        old: auditLog.diff.members.old?.map(
          (groupMember: GroupMember) => groupMember.user_id,
        ),
        new: auditLog.diff.members.new?.map(
          (groupMember: GroupMember) => groupMember.user_id,
        ),
        secret: auditLog.diff.members.secret,
      },
    }
  }

  const toggle = () => {
    if (shouldDisplayDiff) {
      setIsDiffOpen((v) => !v)
    }
  }

  return (
    <TimelineEntry
      key={auditLog.id}
      data-testid={`audit-log-row-${auditLog.id}`}
      clickable={shouldDisplayDiff}
    >
      <TableCell className={styles.auditLogCell}>
        <Stack
          direction="row"
          alignItems="center"
          className={styles.auditLogHeader}
          tabIndex={0}
          onClick={toggle}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              toggle()
            }
          }}
        >
          <Stack
            direction="row"
            alignItems="center"
            className={styles.auditLogHeaderInfo}
          >
            <Stack
              direction="row"
              alignItems="center"
              className={styles.fullWidth}
            >
              <UserAvatar
                username={auditLog.user?.username ?? ""}
                avatarURL={auditLog.user?.avatar_url}
              />

              <Stack
                alignItems="baseline"
                className={styles.fullWidth}
                justifyContent="space-between"
                direction="row"
              >
                <Stack
                  className={styles.auditLogSummary}
                  direction="row"
                  alignItems="baseline"
                  spacing={1}
                >
                  <AuditLogDescription auditLog={auditLog} />
                  <span className={styles.auditLogTime}>
                    {new Date(auditLog.time).toLocaleTimeString()}
                  </span>
                </Stack>

                <Stack direction="row" alignItems="center">
                  <Stack direction="row" spacing={1} alignItems="baseline">
                    <span className={styles.auditLogInfo}>
                      <>{t("auditLog:table.logRow.ip")}</>
                      <strong>
                        {auditLog.ip
                          ? auditLog.ip
                          : t("auditLog:table.logRow.notAvailable")}
                      </strong>
                    </span>

                    <span className={styles.auditLogInfo}>
                      <>{t("auditLog:table.logRow.os")}</>
                      <strong>
                        {os.name
                          ? os.name
                          : // https://github.com/i18next/next-i18next/issues/1795
                            t<string>("auditLog:table.logRow.notAvailable")}
                      </strong>
                    </span>

                    <span className={styles.auditLogInfo}>
                      <>{t("auditLog:table.logRow.browser")}</>
                      <strong>{displayBrowserInfo}</strong>
                    </span>
                  </Stack>

                  <Pill
                    className={styles.httpStatusPill}
                    type={httpStatusColor(auditLog.status_code)}
                    text={auditLog.status_code.toString()}
                  />
                </Stack>
              </Stack>
            </Stack>
          </Stack>

          {shouldDisplayDiff ? (
            <div> {isDiffOpen ? <CloseDropdown /> : <OpenDropdown />}</div>
          ) : (
            <div className={styles.columnWithoutDiff}></div>
          )}
        </Stack>

        {shouldDisplayDiff && (
          <Collapse in={isDiffOpen}>
            <AuditLogDiff diff={auditDiff} />
          </Collapse>
        )}
      </TableCell>
    </TimelineEntry>
  )
}

const useStyles = makeStyles((theme) => ({
  auditLogCell: {
    padding: "0 !important",
    border: 0,
  },

  auditLogHeader: {
    padding: theme.spacing(2, 4),
  },

  auditLogHeaderInfo: {
    flex: 1,
  },

  auditLogSummary: {
    ...theme.typography.body1,
    fontFamily: "inherit",
  },

  auditLogTime: {
    color: theme.palette.text.secondary,
    fontSize: 12,
  },

  auditLogInfo: {
    ...theme.typography.body2,
    fontSize: 12,
    fontFamily: "inherit",
    color: theme.palette.text.secondary,
    display: "block",
  },

  // offset the absence of the arrow icon on diff-less logs
  columnWithoutDiff: {
    marginLeft: "24px",
  },

  fullWidth: {
    width: "100%",
  },

  httpStatusPill: {
    fontSize: 10,
    height: 20,
    paddingLeft: 10,
    paddingRight: 10,
    fontWeight: 600,
  },
}))
