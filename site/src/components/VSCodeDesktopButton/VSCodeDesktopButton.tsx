import { makeStyles } from "@material-ui/core/styles"
import Button from "@material-ui/core/Button"
import { getApiKey } from "api/api"
import { VSCodeIcon } from "components/Icons/VSCodeIcon"
import { FC, PropsWithChildren, useState } from "react"

export interface VSCodeDesktopButtonProps {
  userName: string
  workspaceName: string
  agentName?: string
  folderPath?: string
}

export const VSCodeDesktopButton: FC<
  PropsWithChildren<VSCodeDesktopButtonProps>
> = ({ userName, workspaceName, agentName, folderPath }) => {
  const [loading, setLoading] = useState(false)
  const styles = useStyles()

  return (
    <Button
      startIcon={<VSCodeIcon />}
      disabled={loading}
      className={styles.button}
      onClick={() => {
        setLoading(true)
        getApiKey()
          .then(({ key }) => {
            const query = new URLSearchParams({
              owner: userName,
              workspace: workspaceName,
              url: location.origin,
              token: key,
            })
            if (agentName) {
              query.set("agent", agentName)
            }
            if (folderPath) {
              query.set("folder", folderPath)
            }

            location.href = `vscode://coder.coder-remote/open?${query.toString()}`
          })
          .catch((ex) => {
            console.error(ex)
          })
          .finally(() => {
            setLoading(false)
          })
      }}
    >
      VS Code Desktop
    </Button>
  )
}

const useStyles = makeStyles((theme) => ({
  button: {
    whiteSpace: "nowrap",
    backgroundColor: theme.palette.background.default,
    padding: theme.spacing(0, 3),
    height: 44,
    borderRadius: 6,

    "&:hover": {
      backgroundColor: `${theme.palette.background.paper} !important`,
    },

    "& .MuiButton-startIcon": {
      marginRight: theme.spacing(1.5),
      width: 16,
      height: 16,

      "& svg": {
        width: "100%",
        height: "100%",
      },
    },
  },
}))
