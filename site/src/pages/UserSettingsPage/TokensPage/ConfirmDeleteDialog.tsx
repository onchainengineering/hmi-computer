import type { FC } from "react";
import { getErrorMessage } from "api/errors";
import type { APIKeyWithOwner } from "api/typesGenerated";
import { ConfirmDialog } from "components/Dialogs/ConfirmDialog/ConfirmDialog";
import { displaySuccess, displayError } from "components/GlobalSnackbar/utils";
import { useDeleteToken } from "./hooks";

export interface ConfirmDeleteDialogProps {
  queryKey: (string | boolean)[];
  token: APIKeyWithOwner | undefined;
  setToken: (arg: APIKeyWithOwner | undefined) => void;
}

export const ConfirmDeleteDialog: FC<ConfirmDeleteDialogProps> = ({
  queryKey,
  token,
  setToken,
}) => {
  const tokenName = token?.token_name;

  const { mutate: deleteToken, isLoading: isDeleting } =
    useDeleteToken(queryKey);

  const onDeleteSuccess = () => {
    displaySuccess("Token has been deleted");
    setToken(undefined);
  };

  const onDeleteError = (error: unknown) => {
    const message = getErrorMessage(error, "Failed to delete token");
    displayError(message);
    setToken(undefined);
  };

  return (
    <ConfirmDialog
      type="delete"
      title="Delete Token"
      description={
        <>
          Are you sure you want to permanently delete token{" "}
          <strong>{tokenName}</strong>?
        </>
      }
      open={Boolean(token) || isDeleting}
      confirmLoading={isDeleting}
      onConfirm={() => {
        if (!token) {
          return;
        }
        deleteToken(token.id, {
          onError: onDeleteError,
          onSuccess: onDeleteSuccess,
        });
      }}
      onClose={() => {
        setToken(undefined);
      }}
    />
  );
};
