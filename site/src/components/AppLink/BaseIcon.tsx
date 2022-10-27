import { WorkspaceApp } from "api/typesGenerated"
import { FC } from "react"
import ComputerIcon from "@material-ui/icons/Computer"

export const BaseIcon: FC<{ app: WorkspaceApp }> = ({ app }) => {
  return app.icon ? (
    <img alt={`${app.name} Icon`} src={app.icon} />
  ) : (
    <ComputerIcon />
  )
}
