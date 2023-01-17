import Box from "@material-ui/core/Box"
import Button from "@material-ui/core/Button"
import Link from "@material-ui/core/Link"
import { makeStyles } from "@material-ui/core/styles"
import Typography from "@material-ui/core/Typography"
import TextField from "@material-ui/core/TextField"
import GitHubIcon from "@material-ui/icons/GitHub"
import KeyIcon from "@material-ui/icons/VpnKey"
import { Stack } from "components/Stack/Stack"
import { FormikContextType, FormikTouched, useFormik } from "formik"
import { FC, useState } from "react"
import * as Yup from "yup"
import { AuthMethods } from "../../api/typesGenerated"
import { getFormHelpers, onChangeTrimmed } from "../../util/formUtils"
import { LoadingButton } from "./../LoadingButton/LoadingButton"
import { AlertBanner } from "components/AlertBanner/AlertBanner"
import { useTranslation } from "react-i18next"

/**
 * BuiltInAuthFormValues describes a form using built-in (email/password)
 * authentication. This form may not always be present depending on external
 * auth providers available and administrative configurations
 */
interface BuiltInAuthFormValues {
  email: string
  password: string
}

export enum LoginErrors {
  AUTH_ERROR = "authError",
  GET_USER_ERROR = "getUserError",
  CHECK_PERMISSIONS_ERROR = "checkPermissionsError",
  GET_METHODS_ERROR = "getMethodsError",
}

export const Language = {
  emailLabel: "Email",
  passwordLabel: "Password",
  emailInvalid: "Please enter a valid email address.",
  emailRequired: "Please enter an email address.",
  errorMessages: {
    [LoginErrors.AUTH_ERROR]: "Incorrect email or password.",
    [LoginErrors.GET_USER_ERROR]: "Failed to fetch user details.",
    [LoginErrors.CHECK_PERMISSIONS_ERROR]: "Unable to fetch user permissions.",
    [LoginErrors.GET_METHODS_ERROR]: "Unable to fetch auth methods.",
  },
  passwordSignIn: "Sign In",
  githubSignIn: "GitHub",
  oidcSignIn: "OpenID Connect",
}

const validationSchema = Yup.object({
  email: Yup.string()
    .trim()
    .email(Language.emailInvalid)
    .required(Language.emailRequired),
  password: Yup.string(),
})

const useStyles = makeStyles((theme) => ({
  root: {
    width: "100%",
  },
  title: {
    fontSize: theme.spacing(4),
    fontWeight: 400,
    margin: 0,
    marginBottom: theme.spacing(4),
    lineHeight: 1,

    "& strong": {
      fontWeight: 600,
    },
  },
  buttonIcon: {
    width: 14,
    height: 14,
  },
  divider: {
    paddingTop: theme.spacing(3),
    paddingBottom: theme.spacing(3),
    display: "flex",
    alignItems: "center",
    gap: theme.spacing(2),
  },
  dividerLine: {
    width: "100%",
    height: 1,
    backgroundColor: theme.palette.divider,
  },
  dividerLabel: {
    flexShrink: 0,
    color: theme.palette.text.secondary,
    textTransform: "uppercase",
    fontSize: 12,
    letterSpacing: 1,
  },
  showPasswordLink: {
    cursor: "pointer",
    fontSize: 12,
    color: theme.palette.text.secondary,
    marginTop: 12,
  },
}))

export interface SignInFormProps {
  isLoading: boolean
  redirectTo: string
  loginErrors: Partial<Record<LoginErrors, Error | unknown>>
  authMethods?: AuthMethods
  onSubmit: (credentials: { email: string; password: string }) => void
  // initialTouched is only used for testing the error state of the form.
  initialTouched?: FormikTouched<BuiltInAuthFormValues>
}

export const SignInForm: FC<React.PropsWithChildren<SignInFormProps>> = ({
  authMethods,
  redirectTo,
  isLoading,
  loginErrors,
  onSubmit,
  initialTouched,
}) => {
  const nonPasswordAuthEnabled =
    authMethods?.github.enabled || authMethods?.oidc.enabled

  const [showPasswordAuth, setShowPasswordAuth] = useState(
    !nonPasswordAuthEnabled,
  )
  const styles = useStyles()
  const form: FormikContextType<BuiltInAuthFormValues> =
    useFormik<BuiltInAuthFormValues>({
      initialValues: {
        email: "",
        password: "",
      },
      validationSchema,
      // The email field has an autoFocus, but users may login with a button click.
      // This is set to `false` in order to keep the autoFocus, validateOnChange
      // and Formik experience friendly. Validation will kick in onChange (any
      // field), or after a submission attempt.
      validateOnBlur: false,
      onSubmit,
      initialTouched,
    })
  const getFieldHelpers = getFormHelpers<BuiltInAuthFormValues>(
    form,
    loginErrors.authError,
  )
  const commonTranslation = useTranslation("common")
  const loginPageTranslation = useTranslation("loginPage")

  return (
    <div className={styles.root}>
      <h1 className={styles.title}>
        {loginPageTranslation.t("signInTo")}{" "}
        <strong>{commonTranslation.t("coder")}</strong>
      </h1>
      {showPasswordAuth && (
        <form onSubmit={form.handleSubmit}>
          <Stack>
            {Object.keys(loginErrors).map(
              (errorKey: string) =>
                Boolean(loginErrors[errorKey as LoginErrors]) && (
                  <AlertBanner
                    key={errorKey}
                    severity="error"
                    error={loginErrors[errorKey as LoginErrors]}
                    text={Language.errorMessages[errorKey as LoginErrors]}
                  />
                ),
            )}
            <TextField
              {...getFieldHelpers("email")}
              onChange={onChangeTrimmed(form)}
              autoFocus
              autoComplete="email"
              fullWidth
              label={Language.emailLabel}
              type="email"
              variant="outlined"
            />
            <TextField
              {...getFieldHelpers("password")}
              autoComplete="current-password"
              fullWidth
              id="password"
              label={Language.passwordLabel}
              type="password"
              variant="outlined"
            />
            <div>
              <LoadingButton
                loading={isLoading}
                fullWidth
                type="submit"
                variant="contained"
              >
                {isLoading ? "" : Language.passwordSignIn}
              </LoadingButton>
            </div>
          </Stack>
        </form>
      )}
      {showPasswordAuth && nonPasswordAuthEnabled && (
        <div className={styles.divider}>
          <div className={styles.dividerLine} />
          <div className={styles.dividerLabel}>Or</div>
          <div className={styles.dividerLine} />
        </div>
      )}
      {nonPasswordAuthEnabled && (
        <Box display="grid" gridGap="16px">
          {authMethods.github.enabled && (
            <Link
              underline="none"
              href={`/api/v2/users/oauth2/github/callback?redirect=${encodeURIComponent(
                redirectTo,
              )}`}
            >
              <Button
                startIcon={<GitHubIcon className={styles.buttonIcon} />}
                disabled={isLoading}
                fullWidth
                type="submit"
                variant="contained"
              >
                {Language.githubSignIn}
              </Button>
            </Link>
          )}

          {authMethods.oidc.enabled && (
            <Link
              underline="none"
              href={`/api/v2/users/oidc/callback?redirect=${encodeURIComponent(
                redirectTo,
              )}`}
            >
              <Button
                startIcon={
                  authMethods.oidc.iconUrl ? (
                    <img
                      alt="Open ID Connect icon"
                      src={authMethods.oidc.iconUrl}
                      width="24"
                      height="24"
                    />
                  ) : (
                    <KeyIcon className={styles.buttonIcon} />
                  )
                }
                disabled={isLoading}
                fullWidth
                type="submit"
                variant="contained"
              >
                {authMethods.oidc.signInText || Language.oidcSignIn}
              </Button>
            </Link>
          )}
        </Box>
      )}
      {!showPasswordAuth && (
        <Typography
          className={styles.showPasswordLink}
          onClick={() => setShowPasswordAuth(true)}
        >
          Show password login
        </Typography>
      )}
    </div>
  )
}
