import Button from "@material-ui/core/Button"
import Collapse from "@material-ui/core/Collapse"
import IconButton from "@material-ui/core/IconButton"
import { darken, lighten, makeStyles, Theme } from "@material-ui/core/styles"
import CloseIcon from "@material-ui/icons/Close"
import RefreshIcon from "@material-ui/icons/Refresh"
import { ApiError, getErrorDetail, getErrorMessage } from "api/errors"
import { Expander } from "components/Expander/Expander"
import { Stack } from "components/Stack/Stack"
import { FC, useState } from "react"

export const Language = {
  retryMessage: "Retry",
  unknownErrorMessage: "An unknown error has occurred",
  moreDetails: "More",
  lessDetails: "Less",
}

export interface ErrorSummaryProps {
  error: ApiError | Error | unknown | JSX.Element
  retry?: () => void
  dismissible?: boolean
  defaultMessage?: string
}

export const ErrorSummary: FC<React.PropsWithChildren<ErrorSummaryProps>> = ({
  error,
  retry,
  dismissible,
  defaultMessage,
}) => {
  const message = getErrorMessage(error, defaultMessage || Language.unknownErrorMessage)
  const detail = getErrorDetail(error)
  const [showDetails, setShowDetails] = useState(false)
  const [isOpen, setOpen] = useState(true)

  const styles = useStyles({ showDetails })

  const closeError = () => {
    setOpen(false)
  }

  if (!isOpen) {
    return null
  }

  return (
    <Stack className={styles.root}>
      <Stack direction="row" alignItems="center" className={styles.messageBox}>
        <Stack direction="row" spacing={0}>
          <span className={styles.errorMessage}>{message}</span>
          {!!detail && <Expander expanded={showDetails} setExpanded={setShowDetails} />}
        </Stack>
        {dismissible && (
          <IconButton onClick={closeError} className={styles.iconButton}>
            <CloseIcon className={styles.closeIcon} />
          </IconButton>
        )}
      </Stack>
      <Collapse in={showDetails}>
        <div className={styles.details}>{detail}</div>
      </Collapse>
      {retry && (
        <div className={styles.retry}>
          <Button
            size="small"
            onClick={retry}
            startIcon={<RefreshIcon />}
            variant="outlined"
            className={styles.retryButton}
          >
            {Language.retryMessage}
          </Button>
        </div>
      )}
    </Stack>
  )
}

interface StyleProps {
  showDetails?: boolean
}

const useStyles = makeStyles<Theme, StyleProps>((theme) => ({
  root: {
    background: darken(theme.palette.error.main, 0.6),
    padding: `${theme.spacing(2)}px`,
    borderRadius: theme.shape.borderRadius,
    gap: 0,
  },
  flex: {
    display: "flex",
  },
  messageBox: {
    justifyContent: "space-between",
  },
  errorMessage: {
    marginRight: `${theme.spacing(1)}px`,
  },
  detailsLink: {
    cursor: "pointer",
    color: `${lighten(theme.palette.primary.light, 0.2)}`,
  },
  details: {
    marginTop: `${theme.spacing(2)}px`,
    padding: `${theme.spacing(2)}px`,
    background: darken(theme.palette.error.main, 0.7),
    borderRadius: theme.shape.borderRadius,
  },
  iconButton: {
    padding: 0,
  },
  closeIcon: {
    width: 25,
    height: 25,
    color: theme.palette.primary.contrastText,
  },
  retry: {
    marginTop: `${theme.spacing(2)}px`,
  },
  retryButton: {
    color: theme.palette.error.contrastText,
    borderColor: theme.palette.error.contrastText,

    "&:hover": {
      backgroundColor: theme.palette.error.dark,
    },
  },
}))
