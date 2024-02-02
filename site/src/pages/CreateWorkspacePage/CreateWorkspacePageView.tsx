import { type Interpolation, type Theme } from "@emotion/react";
import TextField from "@mui/material/TextField";
import Button from "@mui/material/Button";
import FormHelperText from "@mui/material/FormHelperText";
import { FormikContextType, useFormik } from "formik";
import { type FC, useEffect, useState, useMemo, useCallback } from "react";
import { useSearchParams } from "react-router-dom";
import * as Yup from "yup";
import type * as TypesGen from "api/typesGenerated";
import {
  getFormHelpers,
  nameValidator,
  onChangeTrimmed,
} from "utils/formUtils";
import {
  FormFields,
  FormSection,
  FormFooter,
  HorizontalForm,
} from "components/Form/Form";
import { UserAutocomplete } from "components/UserAutocomplete/UserAutocomplete";
import {
  AutofillBuildParameter,
  AutofillSource,
  getInitialRichParameterValues,
  useValidationSchemaForRichParameters,
} from "utils/richParameters";
import { ErrorAlert } from "components/Alert/ErrorAlert";
import { Stack } from "components/Stack/Stack";
import { Alert } from "components/Alert/Alert";
import { Margins } from "components/Margins/Margins";
import { Avatar } from "components/Avatar/Avatar";
import {
  PageHeader,
  PageHeaderTitle,
  PageHeaderSubtitle,
} from "components/PageHeader/PageHeader";
import { Pill } from "components/Pill/Pill";
import { RichParameterInput } from "components/RichParameterInput/RichParameterInput";
import { generateWorkspaceName } from "modules/workspaces/generateWorkspaceName";
import {
  CreateWorkspaceMode,
  ExternalAuthPollingState,
} from "./CreateWorkspacePage";
import { ExternalAuthBanner } from "./ExternalAuthBanner/ExternalAuthBanner";
import { CreateWSPermissions } from "./permissions";

export const Language = {
  duplicationWarning:
    "Duplicating a workspace only copies its parameters. No state from the old workspace is copied over.",
} as const;

export interface CreateWorkspacePageViewProps {
  mode: CreateWorkspaceMode;
  error: unknown;
  resetMutation: () => void;
  defaultName?: string | null;
  defaultOwner: TypesGen.User;
  template: TypesGen.Template;
  versionId?: string;
  externalAuth: TypesGen.TemplateVersionExternalAuth[];
  externalAuthPollingState: ExternalAuthPollingState;
  startPollingExternalAuth: () => void;
  parameters: TypesGen.TemplateVersionParameter[];
  autofillParameters: AutofillBuildParameter[];
  permissions: CreateWSPermissions;
  creatingWorkspace: boolean;
  onCancel: () => void;
  onSubmit: (
    req: TypesGen.CreateWorkspaceRequest,
    owner: TypesGen.User,
  ) => void;
}

export const CreateWorkspacePageView: FC<CreateWorkspacePageViewProps> = ({
  mode,
  error,
  resetMutation,
  defaultName,
  defaultOwner,
  template,
  versionId,
  externalAuth,
  externalAuthPollingState,
  startPollingExternalAuth,
  parameters,
  autofillParameters,
  permissions,
  creatingWorkspace,
  onSubmit,
  onCancel,
}) => {
  const [owner, setOwner] = useState(defaultOwner);
  const [searchParams] = useSearchParams();
  const disabledParamsList = searchParams?.get("disable_params")?.split(",");
  const requiresExternalAuth = externalAuth.some((auth) => !auth.authenticated);
  const [suggestedName, setSuggestedName] = useState(() =>
    generateWorkspaceName(),
  );

  const rerollSuggestedName = useCallback(() => {
    setSuggestedName(() => generateWorkspaceName());
  }, []);

  const form: FormikContextType<TypesGen.CreateWorkspaceRequest> =
    useFormik<TypesGen.CreateWorkspaceRequest>({
      initialValues: {
        name: defaultName ?? "",
        template_id: template.id,
        rich_parameter_values: getInitialRichParameterValues(
          parameters,
          autofillParameters,
        ),
      },
      validationSchema: Yup.object({
        name: nameValidator("Workspace Name"),
        rich_parameter_values: useValidationSchemaForRichParameters(parameters),
      }),
      enableReinitialize: true,
      onSubmit: (request) => {
        if (requiresExternalAuth) {
          return;
        }

        onSubmit(request, owner);
      },
    });

  useEffect(() => {
    if (error) {
      window.scrollTo(0, 0);
    }
  }, [error]);

  const getFieldHelpers = getFormHelpers<TypesGen.CreateWorkspaceRequest>(
    form,
    error,
  );

  const autofillSources = useMemo(() => {
    return autofillParameters.reduce(
      (acc, param) => {
        acc[param.name] = param.source;
        return acc;
      },
      {} as Record<string, AutofillSource>,
    );
  }, [autofillParameters]);

  return (
    <Margins size="medium">
      <PageHeader actions={<Button onClick={onCancel}>Cancel</Button>}>
        <Stack direction="row" spacing={3} alignItems="center">
          {template.icon !== "" ? (
            <Avatar size="xl" src={template.icon} variant="square" fitImage />
          ) : (
            <Avatar size="xl">{template.name}</Avatar>
          )}

          <div>
            <PageHeaderTitle>
              {template.display_name.length > 0
                ? template.display_name
                : template.name}
            </PageHeaderTitle>

            <PageHeaderSubtitle condensed>New workspace</PageHeaderSubtitle>
          </div>

          {template.deprecated && <Pill type="warning">Deprecated</Pill>}
        </Stack>
      </PageHeader>

      {requiresExternalAuth ? (
        <ExternalAuthBanner
          providers={externalAuth}
          pollingState={externalAuthPollingState}
          onStartPolling={startPollingExternalAuth}
        />
      ) : (
        <HorizontalForm
          name="create-workspace-form"
          onSubmit={form.handleSubmit}
          css={{ padding: "16px 0" }}
        >
          {Boolean(error) && <ErrorAlert error={error} />}

          {mode === "duplicate" && (
            <Alert severity="info" dismissible>
              {Language.duplicationWarning}
            </Alert>
          )}

          {/* General info */}
          <FormSection
            title="General"
            description={
              permissions.createWorkspaceForUser
                ? "The name of the workspace and its owner. Only admins can create workspace for other users."
                : "The name of your new workspace."
            }
          >
            <FormFields>
              {versionId && versionId !== template.active_version_id && (
                <Stack spacing={1} css={styles.hasDescription}>
                  <TextField
                    disabled
                    fullWidth
                    value={versionId}
                    label="Version ID"
                  />
                  <span css={styles.description}>
                    This parameter has been preset, and cannot be modified.
                  </span>
                </Stack>
              )}
              <div>
                <TextField
                  {...getFieldHelpers("name")}
                  disabled={creatingWorkspace}
                  // resetMutation facilitates the clearing of validation errors
                  onChange={onChangeTrimmed(form, resetMutation)}
                  fullWidth
                  label="Workspace Name"
                />
                <FormHelperText>
                  Need a suggestion?{" "}
                  <Button
                    variant="text"
                    css={styles.nameSuggestion}
                    onClick={async () => {
                      await form.setFieldValue("name", suggestedName);
                      rerollSuggestedName();
                    }}
                  >
                    {suggestedName}
                  </Button>
                </FormHelperText>
              </div>

              {permissions.createWorkspaceForUser && (
                <UserAutocomplete
                  value={owner}
                  onChange={(user) => {
                    setOwner(user ?? defaultOwner);
                  }}
                  label="Owner"
                  size="medium"
                />
              )}
            </FormFields>
          </FormSection>

          {parameters.length > 0 && (
            <FormSection
              title="Parameters"
              description="These are the settings used by your template. Please note that immutable parameters cannot be modified once the workspace is created."
            >
              {/*
                Opted not to use FormFields in order to increase spacing.
                This decision was made because rich parameter inputs are more visually dense than standard text fields.
              */}
              <div css={{ display: "flex", flexDirection: "column", gap: 36 }}>
                {parameters.map((parameter, index) => {
                  const parameterField = `rich_parameter_values.${index}`;
                  const parameterInputName = `${parameterField}.value`;
                  const isDisabled =
                    disabledParamsList?.includes(
                      parameter.name.toLowerCase().replace(/ /g, "_"),
                    ) || creatingWorkspace;

                  return (
                    <RichParameterInput
                      {...getFieldHelpers(parameterInputName)}
                      onChange={async (value) => {
                        await form.setFieldValue(parameterField, {
                          name: parameter.name,
                          value,
                        });
                      }}
                      autofillSource={autofillSources[parameter.name]}
                      key={parameter.name}
                      parameter={parameter}
                      disabled={isDisabled}
                    />
                  );
                })}
              </div>
            </FormSection>
          )}

          <FormFooter
            onCancel={onCancel}
            isLoading={creatingWorkspace}
            submitLabel="Create Workspace"
          />
        </HorizontalForm>
      )}
    </Margins>
  );
};

const styles = {
  nameSuggestion: (theme) => ({
    color: theme.roles.info.fill.solid,
    padding: "4px 8px",
    lineHeight: "inherit",
    fontSize: "inherit",
    height: "unset",
  }),
  hasDescription: {
    paddingBottom: 16,
  },
  description: (theme) => ({
    fontSize: 13,
    color: theme.palette.text.secondary,
  }),
} satisfies Record<string, Interpolation<Theme>>;
