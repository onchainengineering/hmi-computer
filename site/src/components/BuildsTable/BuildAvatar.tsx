import Avatar from "@material-ui/core/Avatar"
import Badge from "@material-ui/core/Badge"
import { Theme, useTheme, withStyles } from "@material-ui/core/styles"
import { FC } from "react"
import PlayArrowOutlined from "@material-ui/icons/PlayArrowOutlined"
import PauseOutlined from "@material-ui/icons/PauseOutlined"
import DeleteOutlined from "@material-ui/icons/DeleteOutlined"
import { WorkspaceBuild, WorkspaceTransition } from "api/typesGenerated"
import { getDisplayWorkspaceBuildStatus } from "util/workspace"
import { PaletteIndex } from "theme/palettes"

interface StylesBadgeProps {
  type: PaletteIndex
}

const StyledBadge = withStyles((theme) => ({
  badge: {
    backgroundColor: ({ type }: StylesBadgeProps) => theme.palette[type].light,
    borderRadius: "100%",
    width: 8,
    minWidth: 8,
    height: 8,
    display: "block",
    padding: 0,
  },
}))(Badge)

const StyledAvatar = withStyles((theme) => ({
  root: {
    background: theme.palette.divider,
    color: theme.palette.text.primary,
    border: `2px solid ${theme.palette.divider}`,
    width: ({ size }: { size?: number }) => size,
    height: ({ size }: { size?: number }) => size,

    "& svg": {
      width: ({ size }: { size?: number }) => (size ? size / 2 : 18),
      height: ({ size }: { size?: number }) => (size ? size / 2 : 18),
    },
  },
}))(Avatar)

export type BuildAvatarProps = {
  build: WorkspaceBuild
  size?: number
}

const iconByTransition: Record<WorkspaceTransition, JSX.Element> = {
  start: <PlayArrowOutlined />,
  stop: <PauseOutlined />,
  delete: <DeleteOutlined />,
}

export const BuildAvatar: FC<BuildAvatarProps> = ({ build, size }) => {
  const theme = useTheme<Theme>()
  const displayBuildStatus = getDisplayWorkspaceBuildStatus(theme, build)

  return (
    <StyledBadge
      role="status"
      type={displayBuildStatus.type}
      arial-label={displayBuildStatus.status}
      title={displayBuildStatus.status}
      overlap="circular"
      anchorOrigin={{
        vertical: "bottom",
        horizontal: "right",
      }}
      badgeContent={<div></div>}
    >
      <StyledAvatar size={size}>
        {iconByTransition[build.transition]}
      </StyledAvatar>
    </StyledBadge>
  )
}
