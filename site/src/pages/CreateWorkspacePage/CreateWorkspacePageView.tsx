import { makeStyles } from "@material-ui/core/styles"
import TextField from "@material-ui/core/TextField"
import { FormikContextType, useFormik } from "formik"
import React from "react"
import * as Yup from "yup"
import * as TypesGen from "../../api/typesGenerated"
import { FormFooter } from "../../components/FormFooter/FormFooter"
import { FullPageForm } from "../../components/FullPageForm/FullPageForm"
import { Margins } from "../../components/Margins/Margins"
import { ParameterInput } from "../../components/ParameterInput/ParameterInput"
import { getFormHelpers, onChangeTrimmed } from "../../util/formUtils"

export const Language = {
  nameRequired: "Please enter a name.",
}

export interface CreateWorkspacePageViewProps {
  loading?: boolean
  template: TypesGen.Template
  templateSchema: TypesGen.ParameterSchema[]

  onCancel: () => void
  onSubmit: (req: TypesGen.CreateWorkspaceRequest) => Promise<void>
}

const validationSchema = Yup.object({
  name: Yup.string().required(Language.nameRequired),
})

export const CreateWorkspacePageView: React.FC<CreateWorkspacePageViewProps> = (props) => {
  const styles = useStyles()
  const [parameterValues, setParameterValues] = React.useState<Record<string, string>>({})
  const form: FormikContextType<TypesGen.CreateWorkspaceRequest> = useFormik<TypesGen.CreateWorkspaceRequest>({
    initialValues: {
      name: "",
      template_id: props.template.id,
    },
    validationSchema,
    onSubmit: (request) => {
      const createRequests: TypesGen.CreateParameterRequest[] = []
      props.templateSchema.forEach((schema) => {
        let value = schema.default_source_value
        if (schema.name in parameterValues) {
          value = parameterValues[schema.name]
        }
        createRequests.push({
          name: schema.name,
          destination_scheme: schema.default_destination_scheme,
          source_scheme: schema.default_source_scheme,
          source_value: value,
        })
      })
      return props.onSubmit({
        ...request,
        parameter_values: createRequests,
      })
    },
  })
  const getFieldHelpers = getFormHelpers<TypesGen.CreateWorkspaceRequest>(form)

  return (
    <Margins>
      <FullPageForm title="Create workspace" onCancel={props.onCancel}>
        <form onSubmit={form.handleSubmit}>
          <TextField
            {...getFieldHelpers("name")}
            disabled={form.isSubmitting}
            onChange={onChangeTrimmed(form)}
            autoFocus
            fullWidth
            label="Name"
            variant="outlined"
          />
          {props.templateSchema.length > 0 && (
            <div className={styles.parameters}>
              {props.templateSchema.map((schema) => (
                <ParameterInput
                  disabled={form.isSubmitting}
                  key={schema.id}
                  onChange={(value) => {
                    setParameterValues({
                      ...parameterValues,
                      [schema.name]: value,
                    })
                  }}
                  schema={schema}
                />
              ))}
            </div>
          )}

          <FormFooter onCancel={props.onCancel} isLoading={props.loading || form.isSubmitting} />
        </form>
      </FullPageForm>
    </Margins>
  )
}

const useStyles = makeStyles((theme) => ({
  parameters: {
    paddingTop: theme.spacing(4),
    "& > *": {
      marginBottom: theme.spacing(4),
    },
  },
}))
