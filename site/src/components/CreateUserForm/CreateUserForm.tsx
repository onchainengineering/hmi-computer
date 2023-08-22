import TextField from "@mui/material/TextField"
import { FormikContextType, useFormik } from "formik"
import { FC } from "react"
import * as Yup from "yup"
import * as TypesGen from "../../api/typesGenerated"
import {
  getFormHelpers,
  nameValidator,
  onChangeTrimmed,
} from "../../utils/formUtils"
import { FormFooter } from "../FormFooter/FormFooter"
import { FullPageForm } from "../FullPageForm/FullPageForm"
import { Stack } from "../Stack/Stack"
import { ErrorAlert } from "components/Alert/ErrorAlert"
import { hasApiFieldErrors, isApiError } from "api/errors"
import MenuItem from "@mui/material/MenuItem"
import { makeStyles } from "@mui/styles"
import { Theme } from "@mui/material/styles"
import { Link } from "@mui/material"
import { Link as RouterLink } from "react-router-dom"

export const Language = {
  emailLabel: "Email",
  passwordLabel: "Password",
  usernameLabel: "Username",
  emailInvalid: "Please enter a valid email address.",
  emailRequired: "Please enter an email address.",
  passwordRequired: "Please enter a password.",
  createUser: "Create",
  cancel: "Cancel",
}

export const authMethodLanguage = {
  password: {
    displayName: "Password",
    description: "Use an email address and password to login",
  },
  oidc: {
    displayName: "OpenID Connect",
    description: "Use an OpenID Connect provider for authentication",
  },
  github: {
    displayName: "Github",
    description: "Use Github OAuth for authentication",
  },
  none: {
    displayName: "None",
    description: (
      <>
        Disable authentication for this user (See the{" "}
        <Link
          component={RouterLink}
          target="_blank"
          rel="noopener"
          to="https://coder.com/docs/v2/latest/admin/auth#disable-built-in-authentication"
        >
          documentation
        </Link>{" "}
        for more details)
      </>
    ),
  },
}

export interface CreateUserFormProps {
  onSubmit: (user: TypesGen.CreateUserRequest) => void
  onCancel: () => void
  error?: unknown
  isLoading: boolean
  myOrgId: string
  authMethods?: TypesGen.AuthMethods
}

const validationSchema = Yup.object({
  email: Yup.string()
    .trim()
    .email(Language.emailInvalid)
    .required(Language.emailRequired),
  password: Yup.string().when("login_type", {
    is: "password",
    then: (schema) => schema.required(Language.passwordRequired),
    otherwise: (schema) => schema,
  }),
  username: nameValidator(Language.usernameLabel),
  login_type: Yup.string().oneOf(Object.keys(authMethodLanguage)),
})

export const CreateUserForm: FC<
  React.PropsWithChildren<CreateUserFormProps>
> = ({ onSubmit, onCancel, error, isLoading, myOrgId, authMethods }) => {
  const form: FormikContextType<TypesGen.CreateUserRequest> =
    useFormik<TypesGen.CreateUserRequest>({
      initialValues: {
        email: "",
        password: "",
        username: "",
        organization_id: myOrgId,
        disable_login: false,
        login_type: "",
      },
      validationSchema,
      onSubmit,
    })
  const getFieldHelpers = getFormHelpers<TypesGen.CreateUserRequest>(
    form,
    error,
  )

  const styles = useStyles()
  // This, unfortunately, cannot be an actual component because mui requires
  // that all `MenuItem`s but be direct children of the `Select` the belong to
  const authMethodSelect = (value: keyof typeof authMethodLanguage) => {
    const language = authMethodLanguage[value]
    return (
      <MenuItem key={value} id={"item-" + value} value={value}>
        <Stack spacing={0} maxWidth={400}>
          {language.displayName}
          <span className={styles.labelDescription}>
            {language.description}
          </span>
        </Stack>
      </MenuItem>
    )
  }

  const methods = []
  if (authMethods?.password.enabled) {
    methods.push(authMethodSelect("password"))
  }
  if (authMethods?.oidc.enabled) {
    methods.push(authMethodSelect("oidc"))
  }
  if (authMethods?.github.enabled) {
    methods.push(authMethodSelect("github"))
  }
  methods.push(authMethodSelect("none"))

  return (
    <FullPageForm title="Create user">
      {isApiError(error) && !hasApiFieldErrors(error) && (
        <ErrorAlert error={error} sx={{ mb: 4 }} />
      )}
      <form onSubmit={form.handleSubmit} autoComplete="off">
        <Stack spacing={2.5}>
          <TextField
            {...getFieldHelpers("username")}
            onChange={onChangeTrimmed(form)}
            autoComplete="username"
            autoFocus
            fullWidth
            label={Language.usernameLabel}
          />
          <TextField
            {...getFieldHelpers("email")}
            onChange={onChangeTrimmed(form)}
            autoComplete="email"
            fullWidth
            label={Language.emailLabel}
          />
          <TextField
            {...getFieldHelpers(
              "login_type",
              "Authentication method for this user",
            )}
            select
            id="login_type"
            data-testid="login-type-input"
            value={form.values.login_type}
            label="Login Type"
            onChange={async (e) => {
              if (e.target.value !== "password") {
                await form.setFieldValue("password", "")
              }
              await form.setFieldValue("login_type", e.target.value)
            }}
            SelectProps={{
              renderValue: (selected: unknown) =>
                authMethodLanguage[selected as keyof typeof authMethodLanguage]
                  ?.displayName ?? "",
            }}
          >
            {methods}
          </TextField>
          <TextField
            {...getFieldHelpers(
              "password",
              form.values.login_type === "password"
                ? ""
                : "No password required for this login type",
            )}
            autoComplete="current-password"
            fullWidth
            id="password"
            data-testid="password-input"
            disabled={form.values.login_type !== "password"}
            label={Language.passwordLabel}
            type="password"
          />
        </Stack>
        <FormFooter onCancel={onCancel} isLoading={isLoading} />
      </form>
    </FullPageForm>
  )
}

const useStyles = makeStyles<Theme>((theme) => ({
  labelDescription: {
    fontSize: 14,
    color: theme.palette.text.secondary,
    wordWrap: "normal",
    whiteSpace: "break-spaces",
  },
}))
