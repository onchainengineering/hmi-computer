import {
  CreateTemplateVersionRequest,
  TemplateVersion,
  TemplateVersionVariable,
} from "api/typesGenerated"
import { AlertBanner } from "components/AlertBanner/AlertBanner"
import { Loader } from "components/Loader/Loader"
import { ComponentProps, FC } from "react"
import { TemplateVariablesForm } from "./TemplateVariablesForm"
import { Stack } from "components/Stack/Stack"
import { makeStyles } from "@material-ui/core/styles"
import { useTranslation } from "react-i18next"
import { FullPageHorizontalForm } from "components/FullPageForm/FullPageHorizontalForm"

export interface TemplateVariablesPageViewProps {
  templateVersion?: TemplateVersion
  templateVariables?: TemplateVersionVariable[]
  onSubmit: (data: CreateTemplateVersionRequest) => void
  onCancel: () => void
  isSubmitting: boolean
  errors?: {
    getTemplateDataError?: unknown
    updateTemplateError?: unknown
  }
  initialTouched?: ComponentProps<
    typeof TemplateVariablesForm
  >["initialTouched"]
}

export const TemplateVariablesPageView: FC<TemplateVariablesPageViewProps> = ({
  templateVersion,
  templateVariables,
  onCancel,
  onSubmit,
  isSubmitting,
  errors = {},
  initialTouched,
}) => {
  const classes = useStyles()
  const isLoading =
    !templateVersion &&
    !templateVariables &&
    !errors.getTemplateDataError &&
    !errors.updateTemplateError
  const { t } = useTranslation("templateVariablesPage")

  return (
    <FullPageHorizontalForm title={t("title")} onCancel={onCancel}>
      {Boolean(errors.getTemplateDataError) && (
        <Stack className={classes.errorContainer}>
          <AlertBanner severity="error" error={errors.getTemplateDataError} />
        </Stack>
      )}
      {Boolean(errors.updateTemplateError) && (
        <Stack className={classes.errorContainer}>
          <AlertBanner severity="error" error={errors.updateTemplateError} />
        </Stack>
      )}
      {isLoading && <Loader />}
      {templateVersion && templateVariables && (
        <TemplateVariablesForm
          initialTouched={initialTouched}
          isSubmitting={isSubmitting}
          templateVersion={templateVersion}
          templateVariables={templateVariables}
          onSubmit={onSubmit}
          onCancel={onCancel}
          error={errors.updateTemplateError}
        />
      )}
    </FullPageHorizontalForm>
  )
}

const useStyles = makeStyles((theme) => ({
  errorContainer: {
    marginBottom: theme.spacing(2),
  },
}))
