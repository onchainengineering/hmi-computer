import { makeStyles } from "@material-ui/core/styles"
import Dialog from "@material-ui/core/Dialog"
import DialogContent from "@material-ui/core/DialogContent"
import DialogContentText from "@material-ui/core/DialogContentText"
import DialogTitle from "@material-ui/core/DialogTitle"
import { DialogProps } from "components/Dialogs/Dialog"
import { FC } from "react"
import { getFormHelpers } from "util/formUtils"
import {
  FormFields,
  VerticalForm,
} from "components/HorizontalForm/HorizontalForm"
import {
  TemplateVersionParameter,
  WorkspaceBuildParameter,
} from "api/typesGenerated"
import { RichParameterInput } from "components/RichParameterInput/RichParameterInput"
import { useFormik } from "formik"
import {
  selectInitialRichParametersValues,
  ValidationSchemaForRichParameters,
} from "util/richParameters"
import * as Yup from "yup"
import DialogActions from "@material-ui/core/DialogActions"
import Button from "@material-ui/core/Button"

export type UpdateBuildParametersDialogProps = DialogProps & {
  onClose: () => void
  onUpdate: (buildParameters: WorkspaceBuildParameter[]) => void
  parameters?: TemplateVersionParameter[]
}

export const UpdateBuildParametersDialog: FC<
  UpdateBuildParametersDialogProps
> = ({ parameters, onUpdate, ...dialogProps }) => {
  const styles = useStyles()
  const form = useFormik({
    initialValues: {
      rich_parameter_values: selectInitialRichParametersValues(parameters),
    },
    validationSchema: Yup.object({
      rich_parameter_values: ValidationSchemaForRichParameters(
        "createWorkspacePage",
        parameters,
      ),
    }),
    onSubmit: (values) => {
      onUpdate(values.rich_parameter_values)
    },
  })
  const getFieldHelpers = getFormHelpers(form)

  return (
    <Dialog
      {...dialogProps}
      scroll="body"
      aria-labelledby="update-build-parameters-title"
      maxWidth="xs"
    >
      <DialogTitle
        id="update-build-parameters-title"
        classes={{ root: styles.title }}
      >
        Workspace parameters
      </DialogTitle>
      <DialogContent className={styles.content}>
        <DialogContentText className={styles.info}>
          It looks like the new version has some mandatory parameters that need
          to be filled in to update the workspace.
        </DialogContentText>
        <VerticalForm
          className={styles.form}
          onSubmit={form.handleSubmit}
          id="updateParameters"
        >
          {parameters && parameters.filter((p) => p.mutable).length > 0 && (
            <FormFields>
              {parameters.map((parameter, index) => {
                if (!parameter.mutable) {
                  return <></>
                }

                return (
                  <RichParameterInput
                    {...getFieldHelpers(
                      "rich_parameter_values[" + index + "].value",
                    )}
                    key={parameter.name}
                    parameter={parameter}
                    initialValue=""
                    index={index}
                    onChange={async (value) => {
                      await form.setFieldValue(
                        "rich_parameter_values." + index,
                        {
                          name: parameter.name,
                          value: value,
                        },
                      )
                    }}
                  />
                )
              })}
            </FormFields>
          )}
        </VerticalForm>
      </DialogContent>
      <DialogActions disableSpacing className={styles.dialogActions}>
        <Button
          fullWidth
          type="button"
          variant="outlined"
          onClick={dialogProps.onClose}
        >
          Cancel
        </Button>
        <Button color="primary" fullWidth type="submit" form="updateParameters">
          Update
        </Button>
      </DialogActions>
    </Dialog>
  )
}

const useStyles = makeStyles((theme) => ({
  title: {
    padding: theme.spacing(3, 5),

    "& h2": {
      fontSize: theme.spacing(2.5),
      fontWeight: 400,
    },
  },

  content: {
    padding: theme.spacing(0, 5, 0, 5),
  },

  info: {
    fontSize: theme.spacing(1.75),
    lineHeight: "160%",
    backgroundColor: theme.palette.info.dark,
    color: theme.palette.text.primary,
    border: `1px solid ${theme.palette.info.light}`,
    borderRight: 0,
    borderLeft: 0,
    padding: theme.spacing(3, 5),
    margin: theme.spacing(0, -5),
  },

  form: {
    paddingTop: theme.spacing(5),
  },

  infoTitle: {
    fontSize: theme.spacing(2),
    fontWeight: 600,
    display: "flex",
    alignItems: "center",
    gap: theme.spacing(1),
  },

  warningIcon: {
    color: theme.palette.warning.light,
    fontSize: theme.spacing(1.5),
  },

  formFooter: {
    flexDirection: "column",
  },

  dialogActions: {
    padding: theme.spacing(5),
    flexDirection: "column",
    gap: theme.spacing(1),
  },
}))
