import { shallowEqual, useActor, useMachine, useSelector } from "@xstate/react"
import { FeatureNames } from "api/types"
import { User } from "api/typesGenerated"
import { useOrganizationId } from "hooks/useOrganizationId"
import { FC, useContext, useState } from "react"
import { Helmet } from "react-helmet-async"
import { useNavigate, useParams } from "react-router-dom"
import { pageTitle } from "util/page"
import { createWorkspaceMachine } from "xServices/createWorkspace/createWorkspaceXService"
import { selectFeatureVisibility } from "xServices/entitlements/entitlementsSelectors"
import { XServiceContext } from "xServices/StateContext"
import { CreateWorkspaceErrors, CreateWorkspacePageView } from "./CreateWorkspacePageView"

const CreateWorkspacePage: FC = () => {
  const xServices = useContext(XServiceContext)
  const organizationId = useOrganizationId()
  const { template } = useParams()
  const templateName = template ? template : ""
  const navigate = useNavigate()
  const featureVisibility = useSelector(
    xServices.entitlementsXService,
    selectFeatureVisibility,
    shallowEqual,
  )
  const workspaceQuotaEnabled = featureVisibility[FeatureNames.WorkspaceQuota]
  const [createWorkspaceState, send] = useMachine(createWorkspaceMachine, {
    context: { organizationId, templateName, workspaceQuotaEnabled },
    actions: {
      onCreateWorkspace: (_, event) => {
        navigate(`/@${event.data.owner_name}/${event.data.name}`)
      },
    },
  })

  const {
    templates,
    templateSchema,
    selectedTemplate,
    getTemplateSchemaError,
    getTemplatesError,
    createWorkspaceError,
    permissions,
    workspaceQuota,
    getWorkspaceQuotaError,
  } = createWorkspaceState.context

  const [authState] = useActor(xServices.authXService)
  const { me } = authState.context

  const [owner, setOwner] = useState<User | null>(me ?? null)

  return (
    <>
      <Helmet>
        <title>{pageTitle("Create Workspace")}</title>
      </Helmet>
      <CreateWorkspacePageView
        loadingTemplates={createWorkspaceState.matches("gettingTemplates")}
        loadingTemplateSchema={createWorkspaceState.matches("gettingTemplateSchema")}
        creatingWorkspace={createWorkspaceState.matches("creatingWorkspace")}
        hasTemplateErrors={createWorkspaceState.matches("error")}
        templateName={templateName}
        templates={templates}
        selectedTemplate={selectedTemplate}
        templateSchema={templateSchema}
        workspaceQuota={workspaceQuota}
        createWorkspaceErrors={{
          [CreateWorkspaceErrors.GET_TEMPLATES_ERROR]: getTemplatesError,
          [CreateWorkspaceErrors.GET_TEMPLATE_SCHEMA_ERROR]: getTemplateSchemaError,
          [CreateWorkspaceErrors.CREATE_WORKSPACE_ERROR]: createWorkspaceError,
          [CreateWorkspaceErrors.GET_WORKSPACE_QUOTA_ERROR]: getWorkspaceQuotaError,
        }}
        canCreateForUser={permissions?.createWorkspaceForUser}
        owner={owner}
        setOwner={setOwner}
        onCancel={() => {
          navigate("/templates")
        }}
        onSubmit={(request) => {
          send({
            type: "CREATE_WORKSPACE",
            request,
            owner,
          })
        }}
        onSelectOwner={(owner) => {
          send({
            type: "SELECT_OWNER",
            owner: owner,
          })
        }}
      />
    </>
  )
}

export default CreateWorkspacePage
