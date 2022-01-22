import React from "react"
import { makeStyles } from "@material-ui/core/styles"
import Box from "@material-ui/core/Box"
import { ProjectName } from "./ProjectName"

export interface ProjectIconProps {
  title: string
  icon?: string
  description?: string
  onClick: () => void
}

const useStyles = makeStyles((theme) => ({
  container: {
    boxShadow: theme.shadows[1],
    cursor: "pointer",
    transition: "box-shadow 250ms ease-in-out",
    backgroundColor: "lightgrey",
    flex: "0",
    margin: "1em",
    display: "flex",
    flexDirection: "column",
    justifyContent: "center",
    alignItems: "center",
    border: `1px solid ${theme.palette.primary.main}`,
    borderRadius: "4px",
    "&:hover": {
      boxShadow: theme.shadows[8],
    },
  },
}))
/**
 * <Circle /> is just a placeholder icon for projects that don't have one.
 */
const Circle: React.FC = () => {
  return (
    <Box
      css={{
        width: "96px",
        height: "96px",
        borderRadius: "96px",
        border: "48px solid white",
      }}
    />
  )
}

export const ProjectIcon: React.FC<ProjectIconProps> = ({ icon, title, onClick }) => {
  const styles = useStyles()

  let iconComponent

  if (typeof icon !== "undefined") {
    iconComponent = <img src={icon} width={"128px"} height={"128px"} />
  } else {
    iconComponent = (
      <Box width={"128px"} height={"128px"} style={{ display: "flex", justifyContent: "center", alignItems: "center" }}>
        <Circle />
      </Box>
    )
  }

  return (
    <div className={styles.container} onClick={onClick}>
      {iconComponent}
      <ProjectName>{title}</ProjectName>
    </div>
  )
}
