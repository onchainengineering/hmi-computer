import DialogActions from "@material-ui/core/DialogActions"
import DialogContent from "@material-ui/core/DialogContent"
import DialogContentText from "@material-ui/core/DialogContentText"
import { makeStyles } from "@material-ui/core/styles"
import React from "react"
import * as TypesGen from "../../api/typesGenerated"
import { CodeBlock } from "../CodeBlock/CodeBlock"
import { Dialog, DialogActionButtons, DialogTitle } from "../Dialog/Dialog"

interface ResetPasswordDialogProps {
  open: boolean
  onClose: () => void
  onConfirm: () => void
  user?: TypesGen.User
  newPassword?: string
}

export const ResetPasswordDialog: React.FC<ResetPasswordDialogProps> = ({
  open,
  onClose,
  onConfirm,
  user,
  newPassword,
}) => {
  const styles = useStyles()

  return (
    <Dialog open={open} onClose={onClose}>
      <DialogTitle title="Reset password" />

      <DialogContent>
        <DialogContentText variant="subtitle2">
          You will need to send <strong>{user?.username}</strong> the following password:
        </DialogContentText>

        <DialogContentText component="div">
          <CodeBlock lines={[newPassword ?? ""]} className={styles.codeBlock} />
        </DialogContentText>
      </DialogContent>

      <DialogActions>
        <DialogActionButtons onCancel={onClose} confirmText="Reset password" onConfirm={onConfirm} />
      </DialogActions>
    </Dialog>
  )
}

const useStyles = makeStyles(() => ({
  codeBlock: {
    minHeight: "auto",
    userSelect: "all",
    width: "100%",
  },
}))
