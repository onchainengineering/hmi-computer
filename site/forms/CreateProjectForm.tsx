import Button from "@material-ui/core/Button"
import { makeStyles } from "@material-ui/core/styles"
import { useFormik } from "formik"
import React from "react"
import * as Yup from "yup"

import {
  FormTitle,
  FormSection,
  formTextFieldFactory,
  formDropdownFieldFactory,
  DropdownItem,
} from "../components/Form"
import { LoadingButton } from "../components/Button"
import { Organization, Project, Provisioner, CreateProjectRequest } from "./../api"

export interface CreateProjectFormProps {
  provisioners: Provisioner[]
  organizations: Organization[]
  onSubmit: (request: CreateProjectRequest) => Promise<Project>
  onCancel: () => void
}

const validationSchema = Yup.object({
  provisioner: Yup.string().required("Provisioner is required."),
  organizationId: Yup.string().required("Organization is required."),
  name: Yup.string().required("Name is required"),
})

const FormTextField = formTextFieldFactory<CreateProjectRequest>()
const FormDropdownField = formDropdownFieldFactory<CreateProjectRequest>()

export const CreateProjectForm: React.FC<CreateProjectFormProps> = ({
  provisioners,
  organizations,
  onSubmit,
  onCancel,
}) => {
  const styles = useStyles()

  const form = useFormik<CreateProjectRequest>({
    initialValues: {
      provisioner: provisioners[0].id,
      organizationId: organizations[0].name,
      name: "",
    },
    enableReinitialize: true,
    validationSchema: validationSchema,
    onSubmit: (req) => {
      return onSubmit(req)
    },
  })

  const organizationDropDownItems: DropdownItem[] = organizations.map((org) => {
    return {
      value: org.name,
      name: org.name,
    }
  })

  const provisionerDropDownItems: DropdownItem[] = provisioners.map((provisioner) => {
    return {
      value: provisioner.id,
      name: provisioner.name,
    }
  })

  return (
    <div className={styles.root}>
      <FormTitle title="Create Project" />

      <FormSection title="Name">
        <FormTextField
          form={form}
          formFieldName="name"
          fullWidth
          helperText="A unique name describing your project."
          label="Project Name"
          placeholder={"my-project"}
          required
        />
      </FormSection>

      <FormSection title="Organization">
        <FormDropdownField
          form={form}
          formFieldName={"organizationId"}
          helperText="The organization owning this project."
          items={organizationDropDownItems}
          fullWidth
          select
          required
        />
      </FormSection>

      <FormSection title="Provider">
        <FormDropdownField
          form={form}
          formFieldName={"provisioner"}
          helperText="The backing provisioner for this project."
          items={provisionerDropDownItems}
          fullWidth
          select
          required
        />
      </FormSection>

      <div className={styles.footer}>
        <Button className={styles.button} onClick={onCancel} variant="outlined">
          Cancel
        </Button>
        <LoadingButton
          loading={form.isSubmitting}
          className={styles.button}
          onClick={() => {
            console.log("submit clicked: " + JSON.stringify(form.values))
            form.submitForm()
          }}
          variant="contained"
          color="primary"
          type="submit"
        >
          Submit
        </LoadingButton>
      </div>
    </div>
  )
}

const useStyles = makeStyles(() => ({
  root: {
    maxWidth: "1380px",
    width: "100%",
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
  },
  footer: {
    display: "flex",
    flex: "0",
    flexDirection: "row",
    justifyContent: "center",
    alignItems: "center",
  },
  button: {
    margin: "1em",
  },
}))
