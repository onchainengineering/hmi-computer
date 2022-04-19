import { makeStyles } from "@material-ui/core/styles"
import { fade } from "@material-ui/core/styles/colorManipulator"
import Typography from "@material-ui/core/Typography"
import React from "react"
import { SectionAction } from "../SectionAction/SectionAction"

type SectionLayout = "fixed" | "fluid"

export interface SectionProps {
  title?: React.ReactNode | string
  description?: React.ReactNode
  toolbar?: React.ReactNode
  alert?: React.ReactNode
  layout?: SectionLayout
  children?: React.ReactNode
}

type SectionFC = React.FC<SectionProps> & { Action: typeof SectionAction }

export const Section: SectionFC = ({ title, description, toolbar, alert, children, layout = "fixed" }) => {
  const styles = useStyles({ layout })
  return (
    <div className={styles.root}>
      <div className={styles.inner}>
        {(title || description) && (
          <div className={styles.header}>
            <div>
              {title && <Typography variant="h4">{title}</Typography>}
              {description && typeof description === "string" && (
                <Typography className={styles.description}>{description}</Typography>
              )}
              {description && typeof description !== "string" && (
                <div className={styles.description}>{description}</div>
              )}
            </div>
            {toolbar && <div>{toolbar}</div>}
          </div>
        )}
        {alert && <div className={styles.alert}>{alert}</div>}
        {children}
      </div>
    </div>
  )
}

// Sub-components
Section.Action = SectionAction

const useStyles = makeStyles((theme) => ({
  root: {
    backgroundColor: theme.palette.background.paper,
    boxShadow: `0px 18px 12px 6px ${fade(theme.palette.common.black, 0.02)}`,
    marginBottom: theme.spacing(1),
    padding: theme.spacing(6),
  },
  inner: ({ layout }: { layout: SectionLayout }) => ({
    maxWidth: layout === "fluid" ? "100%" : 500,
  }),
  alert: {
    marginBottom: theme.spacing(1),
  },
  header: {
    marginBottom: theme.spacing(4),
    display: "flex",
    flexDirection: "row",
    justifyContent: "space-between",
  },
  description: {
    color: theme.palette.text.secondary,
    fontSize: 16,
    marginTop: theme.spacing(2),
  },
}))
