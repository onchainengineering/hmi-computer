import Button from "@material-ui/core/Button"
import { lighten, makeStyles } from "@material-ui/core/styles"
//import SvgIcon from "@material-ui/core/SvgIcon"
import React from "react"

export interface HeaderButtonProps {
  //readonly icon: typeof SvgIcon
  readonly text: string
  readonly disabled?: boolean
  readonly onClick?: (event: MouseEvent) => void
}

export const HeaderButton: React.FC<HeaderButtonProps> = (props) => {
  const styles = useStyles()

  const btn = (
    <Button
      className={styles.pageButton}
      variant="contained"
      onClick={(event: React.MouseEvent): void => {
        if (props.onClick) {
          props.onClick(event.nativeEvent)
        }
      }}
      disabled={props.disabled}
      component="button"
    >
      {props.text}
    </Button>
  )

  return btn
}

const useStyles = makeStyles((theme) => ({
  pageButton: {
    whiteSpace: "nowrap",
    backgroundColor: lighten(theme.palette.hero.main, 0.1),
    color: "#B5BFD2",
  },
}))
