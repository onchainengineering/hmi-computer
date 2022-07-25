import { makeStyles } from "@material-ui/core/styles"
import { combineClasses } from "../../util/combineClasses"
import { Stack } from "../Stack/Stack"

export interface PageHeaderProps {
  actions?: JSX.Element
  className?: string
}

export const PageHeader: React.FC<PageHeaderProps> = ({ children, actions, className }) => {
  const styles = useStyles()

  return (
    <div className={combineClasses([styles.root, className])}>
      <hgroup>{children}</hgroup>
      <Stack direction="row" className={styles.actions}>
        {actions}
      </Stack>
    </div>
  )
}

export const PageHeaderTitle: React.FC = ({ children }) => {
  const styles = useStyles()

  return <h1 className={styles.title}>{children}</h1>
}

export const PageHeaderSubtitle: React.FC = ({ children }) => {
  const styles = useStyles()

  return <h2 className={styles.subtitle}>{children}</h2>
}

const useStyles = makeStyles((theme) => ({
  root: {
    display: "flex",
    alignItems: "center",
    paddingTop: theme.spacing(6),
    paddingBottom: theme.spacing(5),

    [theme.breakpoints.down("md")]: {
      flexDirection: "column",
      alignItems: "flex-start",
    },
  },

  title: {
    fontSize: theme.spacing(4),
    fontWeight: 400,
    margin: 0,
    display: "flex",
    alignItems: "center",
    lineHeight: "140%",
  },

  subtitle: {
    fontSize: theme.spacing(2.25),
    color: theme.palette.text.secondary,
    fontWeight: 400,
    display: "block",
    margin: 0,
    marginTop: theme.spacing(1),
  },

  actions: {
    marginLeft: "auto",

    [theme.breakpoints.down("md")]: {
      marginTop: theme.spacing(3),
      marginLeft: "initial",
      width: "100%",
    },
  },
}))
