import type { Interpolation, Theme } from "@emotion/react";
import Checkbox from "@mui/material/Checkbox";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableRow from "@mui/material/TableRow";
import TextField from "@mui/material/TextField";
import type { useFormik } from "formik";
import { type ChangeEvent, useState, type FC } from "react";
import { useNavigate } from "react-router-dom";
import { isApiValidationError } from "api/errors";
import { RBACResourceActions } from "api/rbacresources_gen";
import type {
  Role,
  PatchRoleRequest,
  Permission,
  AssignableRoles,
  RBACResource,
  RBACAction,
} from "api/typesGenerated";
import { ErrorAlert } from "components/Alert/ErrorAlert";
import {
  FormFields,
  FormFooter,
  FormSection,
  HorizontalForm,
} from "components/Form/Form";
import { PageHeader, PageHeaderTitle } from "components/PageHeader/PageHeader";
import { getFormHelpers } from "utils/formUtils";

export type CreateEditRolePageViewProps = {
  role: AssignableRoles | undefined;
  form: ReturnType<typeof useFormik<PatchRoleRequest>>;
  error?: unknown;
  isLoading: boolean;
};

export const CreateEditRolePageView: FC<CreateEditRolePageViewProps> = ({
  role,
  form,
  error,
  isLoading,
}) => {
  const navigate = useNavigate();
  const getFieldHelpers = getFormHelpers<Role>(form, error);
  const onCancel = () => navigate(-1);

  return (
    <>
      <PageHeader css={{ paddingTop: 8 }}>
        <PageHeaderTitle>
          {role ? "Edit" : "Create"} custom role
        </PageHeaderTitle>
      </PageHeader>
      <HorizontalForm onSubmit={form.handleSubmit}>
        <FormSection
          title="Role settings"
          description="Set a name and permissions for this role."
        >
          <FormFields>
            {Boolean(error) && !isApiValidationError(error) && (
              <ErrorAlert error={error} />
            )}

            <TextField
              {...getFieldHelpers("name", {
                helperText:
                  "The role name cannot be modified after the role is created.",
              })}
              autoFocus
              fullWidth
              disabled={role !== undefined}
              label="Name"
            />
            <TextField
              {...getFieldHelpers("display_name", {
                helperText: "Optional: keep empty to default to the name.",
              })}
              fullWidth
              label="Display Name"
            />
            <ActionCheckboxes
              permissions={role?.organization_permissions || []}
              form={form}
            />
          </FormFields>
        </FormSection>
        <FormFooter
          onCancel={onCancel}
          isLoading={isLoading}
          submitLabel={role !== undefined ? "Save" : "Create Role"}
        />
      </HorizontalForm>
    </>
  );
};

interface ActionCheckboxesProps {
  permissions: readonly Permission[] | undefined;
  form: ReturnType<typeof useFormik<Role>> & { values: Role };
}

const ActionCheckboxes: FC<ActionCheckboxesProps> = ({ permissions, form }) => {
  const [checkedActions, setCheckActions] = useState(permissions);

  const handleCheckChange = async (
    e: ChangeEvent<HTMLInputElement>,
    form: ReturnType<typeof useFormik<Role>> & { values: Role },
  ) => {
    const { name, checked } = e.currentTarget;
    const [resource_type, action] = name.split(":");

    const newPermissions = checked
      ? [
          ...(checkedActions ?? []),
          {
            negate: false,
            resource_type: resource_type as RBACResource,
            action: action as RBACAction,
          },
        ]
      : checkedActions?.filter(
          (p) => p.resource_type !== resource_type || p.action !== action,
        );

    setCheckActions(newPermissions);
    await form.setFieldValue("organization_permissions", newPermissions);
  };

  return (
    <TableContainer>
      <Table>
        <TableBody>
          {Object.entries(RBACResourceActions).map(([resourceKey, value]) => {
            return (
              <TableRow key={resourceKey}>
                <TableCell>
                  <li key={resourceKey} css={styles.checkBoxes}>
                    {resourceKey}
                    <ul css={styles.checkBoxes}>
                      {Object.entries(value).map(([actionKey, value]) => {
                        return (
                          <li key={actionKey}>
                            <span css={styles.actionText}>
                              <Checkbox
                                name={`${resourceKey}:${actionKey}`}
                                checked={
                                  checkedActions?.some(
                                    (p) =>
                                      p.resource_type === resourceKey &&
                                      (p.action.toString() === "*" ||
                                        p.action === actionKey),
                                  ) || false
                                }
                                onChange={(e) => handleCheckChange(e, form)}
                              />
                              {actionKey}
                            </span>{" "}
                            -{" "}
                            <span css={styles.actionDescription}>{value}</span>
                          </li>
                        );
                      })}
                    </ul>
                  </li>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

const styles = {
  checkBoxes: {
    margin: 0,
    listStyleType: "none",
  },
  actionText: (theme) => ({
    color: theme.palette.text.primary,
  }),
  actionDescription: (theme) => ({
    color: theme.palette.text.secondary,
  }),
} satisfies Record<string, Interpolation<Theme>>;

export default CreateEditRolePageView;
