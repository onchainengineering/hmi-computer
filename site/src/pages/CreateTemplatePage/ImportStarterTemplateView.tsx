import { useQuery, useMutation } from "@tanstack/react-query";
import { templateVersionLogs } from "api/queries/templateVersions";
import {
  JobError,
  createTemplate,
  templateExamples,
} from "api/queries/templates";
import { ErrorAlert } from "components/Alert/ErrorAlert";
import { useOrganizationId } from "hooks";
import { useNavigate, useSearchParams } from "react-router-dom";
import { CreateTemplateForm } from "./CreateTemplateForm";
import { Loader } from "components/Loader/Loader";
import { useDashboard } from "components/Dashboard/DashboardProvider";
import { firstVersion, getFormPermissions, newTemplate } from "./utils";

export const ImportStarterTemplateView = () => {
  const navigate = useNavigate();
  const organizationId = useOrganizationId();
  const [searchParams] = useSearchParams();
  const templateExamplesQuery = useQuery(templateExamples(organizationId));
  const templateExample = templateExamplesQuery.data?.find(
    (e) => e.id === searchParams.get("exampleId")!,
  );

  const isLoading = templateExamplesQuery.isLoading;
  const loadingError = templateExamplesQuery.error;

  const dashboard = useDashboard();
  const formPermissions = getFormPermissions(dashboard.entitlements);

  const createTemplateMutation = useMutation(createTemplate());
  const createError = createTemplateMutation.error;
  const isJobError = createError instanceof JobError;
  const templateVersionLogsQuery = useQuery({
    ...templateVersionLogs(isJobError ? createError.version.id : ""),
    enabled: isJobError,
  });

  if (isLoading) {
    return <Loader />;
  }

  if (loadingError) {
    return <ErrorAlert error={loadingError} />;
  }

  return (
    <CreateTemplateForm
      {...formPermissions}
      starterTemplate={templateExample!}
      error={createTemplateMutation.error}
      isSubmitting={createTemplateMutation.isLoading}
      onCancel={() => navigate(-1)}
      jobError={isJobError ? createError.job.error : undefined}
      logs={templateVersionLogsQuery.data}
      onSubmit={async (formData) => {
        const template = await createTemplateMutation.mutateAsync({
          organizationId,
          version: firstVersion(
            templateExample!,
            formData.user_variable_values,
          ),
          template: newTemplate(formData),
        });
        navigate(`/templates/${template.name}`);
      }}
    />
  );
};
