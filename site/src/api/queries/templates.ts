import type { MutationOptions, QueryClient, QueryOptions } from "react-query";
import * as API from "api/api";
import type {
  CreateTemplateRequest,
  CreateTemplateVersionRequest,
  ProvisionerJob,
  ProvisionerJobStatus,
  UsersRequest,
  Template,
  TemplateRole,
  TemplateVersion,
} from "api/typesGenerated";
import { delay } from "utils/delay";
import { getTemplateVersionFiles } from "utils/templateVersion";

export const templateByNameKey = (orgId: string, name: string) => [
  orgId,
  "template",
  name,
  "settings",
];

export const templateByName = (
  orgId: string,
  name: string,
): QueryOptions<Template> => {
  return {
    queryKey: templateByNameKey(orgId, name),
    queryFn: async () => API.getTemplateByName(orgId, name),
  };
};

const getTemplatesQueryKey = (orgId: string, deprecated?: boolean) => [
  orgId,
  "templates",
  deprecated,
];

export const templates = (orgId: string, deprecated?: boolean) => {
  return {
    queryKey: getTemplatesQueryKey(orgId, deprecated),
    queryFn: () => API.getTemplates(orgId, { deprecated }),
  };
};

export const templateACL = (templateId: string) => {
  return {
    queryKey: ["templateAcl", templateId],
    queryFn: () => API.getTemplateACL(templateId),
  };
};

export const setUserRole = (
  queryClient: QueryClient,
): MutationOptions<
  Awaited<ReturnType<typeof API.updateTemplateACL>>,
  unknown,
  { templateId: string; userId: string; role: TemplateRole }
> => {
  return {
    mutationFn: ({ templateId, userId, role }) =>
      API.updateTemplateACL(templateId, {
        user_perms: {
          [userId]: role,
        },
      }),
    onSuccess: async (_res, { templateId }) => {
      await queryClient.invalidateQueries(["templateAcl", templateId]);
    },
  };
};

export const setGroupRole = (
  queryClient: QueryClient,
): MutationOptions<
  Awaited<ReturnType<typeof API.updateTemplateACL>>,
  unknown,
  { templateId: string; groupId: string; role: TemplateRole }
> => {
  return {
    mutationFn: ({ templateId, groupId, role }) =>
      API.updateTemplateACL(templateId, {
        group_perms: {
          [groupId]: role,
        },
      }),
    onSuccess: async (_res, { templateId }) => {
      await queryClient.invalidateQueries(["templateAcl", templateId]);
    },
  };
};

export const templateExamples = (orgId: string) => {
  return {
    queryKey: [...getTemplatesQueryKey(orgId), "examples"],
    queryFn: () => API.getTemplateExamples(orgId),
  };
};

export const templateVersion = (versionId: string) => {
  return {
    queryKey: ["templateVersion", versionId],
    queryFn: () => API.getTemplateVersion(versionId),
  };
};

export const templateVersionByName = (
  orgId: string,
  templateName: string,
  versionName: string,
) => {
  return {
    queryKey: ["templateVersion", orgId, templateName, versionName],
    queryFn: () =>
      API.getTemplateVersionByName(orgId, templateName, versionName),
  };
};

export const templateVersions = (templateId: string) => {
  return {
    queryKey: ["templateVersions", templateId],
    queryFn: () => API.getTemplateVersions(templateId),
  };
};

export const templateVersionVariablesKey = (versionId: string) => [
  "templateVersion",
  versionId,
  "variables",
];

export const templateVersionVariables = (versionId: string) => {
  return {
    queryKey: templateVersionVariablesKey(versionId),
    queryFn: () => API.getTemplateVersionVariables(versionId),
  };
};

export const createTemplateVersion = (orgId: string) => {
  return {
    mutationFn: async (request: CreateTemplateVersionRequest) => {
      const newVersion = await API.createTemplateVersion(orgId, request);
      return newVersion;
    },
  };
};

export const createAndBuildTemplateVersion = (orgId: string) => {
  return {
    mutationFn: async (request: CreateTemplateVersionRequest) => {
      const newVersion = await API.createTemplateVersion(orgId, request);
      await waitBuildToBeFinished(newVersion);
      return newVersion;
    },
  };
};

export const updateActiveTemplateVersion = (
  template: Template,
  queryClient: QueryClient,
) => {
  return {
    mutationFn: (versionId: string) =>
      API.updateActiveTemplateVersion(template.id, {
        id: versionId,
      }),
    onSuccess: async () => {
      // invalidated because of `active_version_id`
      await queryClient.invalidateQueries(
        templateByNameKey(template.organization_id, template.name),
      );
    },
  };
};

export const templaceACLAvailable = (
  templateId: string,
  options: UsersRequest,
) => {
  return {
    queryKey: ["template", templateId, "aclAvailable", options],
    queryFn: () => API.getTemplateACLAvailable(templateId, options),
  };
};

export const templateVersionExternalAuthKey = (versionId: string) => [
  "templateVersion",
  versionId,
  "externalAuth",
];

export const templateVersionExternalAuth = (versionId: string) => {
  return {
    queryKey: templateVersionExternalAuthKey(versionId),
    queryFn: () => API.getTemplateVersionExternalAuth(versionId),
  };
};

export const createTemplate = () => {
  return {
    mutationFn: createTemplateFn,
  };
};

export type CreateTemplateOptions = {
  orgId: string;
  version: CreateTemplateVersionRequest;
  template: Omit<CreateTemplateRequest, "template_version_id">;
  onCreateVersion?: (version: TemplateVersion) => void;
  onTemplateVersionChanges?: (version: TemplateVersion) => void;
};

const createTemplateFn = async (options: CreateTemplateOptions) => {
  const version = await API.createTemplateVersion(
    options.orgId,
    options.version,
  );
  options.onCreateVersion?.(version);
  await waitBuildToBeFinished(version, options.onTemplateVersionChanges);
  return API.createTemplate(options.orgId, {
    ...options.template,
    template_version_id: version.id,
  });
};

export const templateVersionLogs = (versionId: string) => {
  return {
    queryKey: ["templateVersion", versionId, "logs"],
    queryFn: () => API.getTemplateVersionLogs(versionId),
  };
};

export const richParameters = (versionId: string) => {
  return {
    queryKey: ["templateVersion", versionId, "richParameters"],
    queryFn: () => API.getTemplateVersionRichParameters(versionId),
  };
};

export const resources = (versionId: string) => {
  return {
    queryKey: ["templateVersion", versionId, "resources"],
    queryFn: () => API.getTemplateVersionResources(versionId),
  };
};

export const templateFiles = (fileId: string) => {
  return {
    queryKey: ["templateFiles", fileId],
    queryFn: async () => {
      const tarFile = await API.getFile(fileId);
      return getTemplateVersionFiles(tarFile);
    },
  };
};

export const previousTemplateVersion = (
  orgId: string,
  templateName: string,
  versionName: string,
) => {
  return {
    queryKey: [
      "templateVersion",
      orgId,
      templateName,
      versionName,
      "previous",
    ],
    queryFn: async () => {
      const result = await API.getPreviousTemplateVersionByName(
        orgId,
        templateName,
        versionName,
      );

      return result ?? null;
    },
  };
};

const waitBuildToBeFinished = async (
  version: TemplateVersion,
  onRequest?: (data: TemplateVersion) => void,
) => {
  let data: TemplateVersion;
  let jobStatus: ProvisionerJobStatus | undefined = undefined;
  do {
    // When pending we want to poll more frequently
    await delay(jobStatus === "pending" ? 250 : 1000);
    data = await API.getTemplateVersion(version.id);
    onRequest?.(data);
    jobStatus = data.job.status;

    if (jobStatus === "succeeded") {
      return version.id;
    }
  } while (jobStatus === "pending" || jobStatus === "running");

  // No longer pending/running, but didn't succeed
  throw new JobError(data.job, version);
};

export class JobError extends Error {
  public job: ProvisionerJob;
  public version: TemplateVersion;

  constructor(job: ProvisionerJob, version: TemplateVersion) {
    super(job.error);
    this.job = job;
    this.version = version;
  }
}
