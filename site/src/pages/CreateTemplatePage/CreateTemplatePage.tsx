import { FullPageHorizontalForm } from "components/FullPageForm/FullPageHorizontalForm";
import { FC } from "react";
import { Helmet } from "react-helmet-async";
import { useNavigate, useSearchParams } from "react-router-dom";
import { pageTitle } from "utils/page";
import { DuplicateTemplateView } from "./DuplicateTemplateView";
import { ImportStarterTemplateView } from "./ImportStarterTemplateView";

const CreateTemplatePage: FC = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  // const organizationId = useOrganizationId();
  // const [state, send] = useMachine(createTemplateMachine, {
  //   context: {
  //     organizationId,
  //     exampleId: searchParams.get("exampleId"),
  //     templateNameToCopy: searchParams.get("fromTemplate"),
  //   },
  //   actions: {
  //     onCreate: (_, { data }) => {
  //       navigate(`/templates/${data.name}`);
  //     },
  //   },
  // });

  // const { starterTemplate, error, file, jobError, jobLogs, variables } =
  //   state.context;
  // const shouldDisplayForm = !state.hasTag("loading");
  // const { entitlements } = useDashboard();
  // const allowAdvancedScheduling =
  //   entitlements.features["advanced_template_scheduling"].enabled;
  // // Requires the template RBAC feature, otherwise disabling everyone access
  // // means no one can access.
  // const allowDisableEveryoneAccess =
  //   entitlements.features["template_rbac"].enabled;
  // const allowAutostopRequirement =
  //   entitlements.features["template_autostop_requirement"].enabled;

  const onCancel = () => {
    navigate(-1);
  };

  return (
    <>
      <Helmet>
        <title>{pageTitle("Create Template")}</title>
      </Helmet>

      <FullPageHorizontalForm title="Create Template" onCancel={onCancel}>
        {searchParams.has("fromTemplate") ? (
          <DuplicateTemplateView />
        ) : searchParams.has("exampleId") ? (
          <ImportStarterTemplateView />
        ) : (
          <UploadTemplateView />
        )}
        {/* {state.hasTag("loading") && <Loader />}

        <Stack spacing={6}>
          {Boolean(error) && !isApiValidationError(error) && (
            <ErrorAlert error={error} />
          )}

          {shouldDisplayForm && (
            <CreateTemplateForm
              copiedTemplate={state.context.copiedTemplate}
              allowAdvancedScheduling={allowAdvancedScheduling}
              allowDisableEveryoneAccess={allowDisableEveryoneAccess}
              allowAutostopRequirement={allowAutostopRequirement}
              error={error}
              starterTemplate={starterTemplate}
              isSubmitting={state.hasTag("submitting")}
              variables={variables}
              onCancel={onCancel}
              onSubmit={(data) => {
                send({
                  type: "CREATE",
                  data,
                });
              }}
              upload={{
                file,
                isUploading: state.matches("uploading"),
                onRemove: () => {
                  send("REMOVE_FILE");
                },
                onUpload: (file) => {
                  send({ type: "UPLOAD_FILE", file });
                },
              }}
              jobError={jobError}
              logs={jobLogs}
            />
          )}
        </Stack> */}
      </FullPageHorizontalForm>
    </>
  );
};

const UploadTemplateView = () => {
  return <div>Upload</div>;
};

export default CreateTemplatePage;
