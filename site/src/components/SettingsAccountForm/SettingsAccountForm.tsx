import TextField from "@mui/material/TextField"
import { FormikContextType, FormikTouched, useFormik } from "formik"
import { FC } from "react"
import * as Yup from "yup"
import {
  getFormHelpers,
  nameValidator,
  onChangeTrimmed,
} from "../../utils/formUtils"
import { LoadingButton } from "../LoadingButton/LoadingButton"
import { Stack } from "../Stack/Stack"
import { Alert } from "components/Alert/Alert"
import { getErrorMessage } from "api/errors"

export interface AccountFormValues {
  username: string
}

export const Language = {
  usernameLabel: "Username",
  emailLabel: "Email",
  updateSettings: "Update settings",
}

const validationSchema = Yup.object({
  username: nameValidator(Language.usernameLabel),
})

export interface AccountFormProps {
  editable: boolean
  email: string
  isLoading: boolean
  initialValues: AccountFormValues
  onSubmit: (values: AccountFormValues) => void
  updateProfileError?: Error | unknown
  // initialTouched is only used for testing the error state of the form.
  initialTouched?: FormikTouched<AccountFormValues>
}

export const AccountForm: FC<React.PropsWithChildren<AccountFormProps>> = ({
  editable,
  email,
  isLoading,
  onSubmit,
  initialValues,
  updateProfileError,
  initialTouched,
}) => {
  const form: FormikContextType<AccountFormValues> =
    useFormik<AccountFormValues>({
      initialValues,
      validationSchema,
      onSubmit,
      initialTouched,
    })
  const getFieldHelpers = getFormHelpers<AccountFormValues>(
    form,
    updateProfileError,
  )

  return (
    <>
      <form onSubmit={form.handleSubmit}>
        <Stack>
          {Boolean(updateProfileError) && (
            <Alert severity="error">
              {getErrorMessage(updateProfileError, "Error updating profile")}
            </Alert>
          )}
          <TextField
            disabled
            fullWidth
            label={Language.emailLabel}
            value={email}
          />
          <TextField
            {...getFieldHelpers("username")}
            onChange={onChangeTrimmed(form)}
            aria-disabled={!editable}
            autoComplete="username"
            disabled={!editable}
            fullWidth
            label={Language.usernameLabel}
          />

          <div>
            <LoadingButton
              loading={isLoading}
              aria-disabled={!editable}
              disabled={!editable}
              type="submit"
              variant="contained"
            >
              {isLoading ? "" : Language.updateSettings}
            </LoadingButton>
          </div>
        </Stack>
      </form>
    </>
  )
}
