import Badge from "@mui/material/Badge"
import { useTheme, withStyles } from "@mui/styles"
import { FC } from "react"
import PlayArrowOutlined from "@mui/icons-material/PlayArrowOutlined"
import PauseOutlined from "@mui/icons-material/PauseOutlined"
import DeleteOutlined from "@mui/icons-material/DeleteOutlined"
import { WorkspaceBuild, WorkspaceTransition } from "api/typesGenerated"
import { getDisplayJobStatus } from "utils/workspace"
import { Avatar, AvatarProps } from "components/Avatar/Avatar"
import { PaletteIndex } from "theme/theme"
import { Theme } from "@mui/material/styles"

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

export interface BuildAvatarProps {
  build: WorkspaceBuild
  size?: AvatarProps["size"]
}

const iconByTransition: Record<WorkspaceTransition, JSX.Element> = {
  start: <PlayArrowOutlined />,
  stop: <PauseOutlined />,
  delete: <DeleteOutlined />,
}

export const BuildAvatar: FC<BuildAvatarProps> = ({ build, size }) => {
  const theme = useTheme<Theme>()
  const displayBuildStatus = getDisplayJobStatus(theme, build.job.status)

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
      <Avatar size={size} colorScheme="darken">
        {iconByTransition[build.transition]}
      </Avatar>
    </StyledBadge>
  )
}
