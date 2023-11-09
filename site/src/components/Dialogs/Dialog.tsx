import MuiDialog, { DialogProps as MuiDialogProps } from "@mui/material/Dialog";
import { type ReactNode } from "react";
import { colors } from "theme/colors";
import { ConfirmDialogType } from "./types";
import { type Interpolation, type Theme } from "@emotion/react";
import LoadingButton, { LoadingButtonProps } from "@mui/lab/LoadingButton";

export interface DialogActionButtonsProps {
  /** Text to display in the cancel button */
  cancelText?: string;
  /** Text to display in the confirm button */
  confirmText?: ReactNode;
  /** Whether or not confirm is loading, also disables cancel when true */
  confirmLoading?: boolean;
  /** Whether or not the submit button is disabled */
  disabled?: boolean;
  /** Called when cancel is clicked */
  onCancel?: () => void;
  /** Called when confirm is clicked */
  onConfirm?: () => void;
  type?: ConfirmDialogType;
}

const typeToColor = (type: ConfirmDialogType): LoadingButtonProps["color"] => {
  if (type === "danger") {
    return "secondary";
  }
  return "primary";
};

/**
 * Quickly handles most modals actions, some combination of a cancel and confirm button
 */
export const DialogActionButtons: React.FC<DialogActionButtonsProps> = ({
  cancelText = "Cancel",
  confirmText = "Confirm",
  confirmLoading = false,
  disabled = false,
  onCancel,
  onConfirm,
  type = "info",
}) => {
  return (
    <>
      {onCancel && (
        <LoadingButton disabled={confirmLoading} onClick={onCancel} fullWidth>
          {cancelText}
        </LoadingButton>
      )}

      {onConfirm && (
        <LoadingButton
          fullWidth
          data-testid="confirm-button"
          variant="contained"
          onClick={onConfirm}
          color={typeToColor(type)}
          loading={confirmLoading}
          disabled={disabled}
          type="submit"
          css={[
            type === "danger" && styles.dangerButton,
            type === "success" && styles.successButton,
          ]}
        >
          {confirmText}
        </LoadingButton>
      )}
    </>
  );
};

const styles = {
  dangerButton: (theme) => ({
    "&.MuiButton-contained": {
      backgroundColor: colors.red[10],
      borderColor: colors.red[9],

      "&:not(.MuiLoadingButton-loading)": {
        color: theme.deprecated.palette.text.primary,
      },

      "&:hover:not(:disabled)": {
        backgroundColor: colors.red[9],
        borderColor: colors.red[9],
      },

      "&.Mui-disabled": {
        backgroundColor: colors.red[15],
        borderColor: colors.red[15],

        "&:not(.MuiLoadingButton-loading)": {
          color: colors.red[9],
        },
      },
    },
  }),
  successButton: (theme) => ({
    "&.MuiButton-contained": {
      backgroundColor: theme.deprecated.palette.success.main,

      "&:not(.MuiLoadingButton-loading)": {
        color: theme.deprecated.palette.primary.contrastText,
      },

      "&:hover": {
        backgroundColor: theme.deprecated.palette.success.dark,

        "@media (hover: none)": {
          backgroundColor: "transparent",
        },

        "&.Mui-disabled": {
          backgroundColor: "transparent",
        },
      },

      "&.Mui-disabled": {
        backgroundColor: theme.deprecated.palette.success.dark,

        "&:not(.MuiLoadingButton-loading)": {
          color: theme.deprecated.palette.text.secondary,
        },
      },
    },

    "&.MuiButton-outlined": {
      color: theme.deprecated.palette.success.main,
      borderColor: theme.deprecated.palette.success.main,
      "&:hover": {
        backgroundColor: theme.deprecated.palette.success.dark,
        "@media (hover: none)": {
          backgroundColor: "transparent",
        },
        "&.Mui-disabled": {
          backgroundColor: "transparent",
        },
      },
      "&.Mui-disabled": {
        color: theme.deprecated.palette.text.secondary,
        borderColor: theme.deprecated.palette.action.disabled,
      },
    },

    "&.MuiButton-text": {
      color: theme.deprecated.palette.success.main,
      "&:hover": {
        backgroundColor: theme.deprecated.palette.success.dark,
        "@media (hover: none)": {
          backgroundColor: "transparent",
        },
      },
      "&.Mui-disabled": {
        color: theme.deprecated.palette.text.secondary,
      },
    },
  }),
} satisfies Record<string, Interpolation<Theme>>;

export type DialogProps = MuiDialogProps;

/**
 * Wrapper around Material UI's Dialog component. Conveniently exports all of
 * Dialog's components in one import, so for example `<DialogContent />` becomes
 * `<Dialog.Content />` etc. Also contains some custom Dialog components listed below.
 *
 * See original component's Material UI documentation here: https://material-ui.com/components/dialogs/
 */
export const Dialog: React.FC<DialogProps> = (props) => {
  // Wrapped so we can add custom attributes below
  return <MuiDialog {...props} />;
};
