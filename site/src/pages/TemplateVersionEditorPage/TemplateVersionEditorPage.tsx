import { useMachine } from "@xstate/react";
import { TemplateVersionEditor } from "./TemplateVersionEditor";
import { useOrganizationId } from "hooks/useOrganizationId";
import { FC, useEffect, useRef, useState } from "react";
import { Helmet } from "react-helmet-async";
import { useNavigate, useParams } from "react-router-dom";
import { pageTitle } from "utils/page";
import { templateVersionEditorMachine } from "xServices/templateVersionEditor/templateVersionEditorXService";
import { useMutation, useQuery } from "react-query";
import { templateByName, templateVersionByName } from "api/queries/templates";
import { file, uploadFile } from "api/queries/files";
import { TarReader, TarWriter } from "utils/tar";
import { FileTree, traverse } from "utils/filetree";
import {
  createTemplateVersionFileTree,
  isAllowedFile,
} from "utils/templateVersion";

type Params = {
  version: string;
  template: string;
};

export const TemplateVersionEditorPage: FC = () => {
  const navigate = useNavigate();
  const { version: versionName, template: templateName } =
    useParams() as Params;
  const orgId = useOrganizationId();
  const templateQuery = useQuery(templateByName(orgId, templateName));
  const templateVersionQuery = useQuery(
    templateVersionByName(orgId, templateName, versionName),
  );
  const fileQuery = useQuery({
    ...file(templateVersionQuery.data?.job.file_id ?? ""),
    enabled: templateVersionQuery.isSuccess,
  });
  const [editorState, sendEvent] = useMachine(templateVersionEditorMachine, {
    context: { orgId, templateId: templateQuery.data?.id },
  });
  const [fileTree, setFileTree] = useState<FileTree>();
  const uploadFileMutation = useMutation(uploadFile());
  const currentTarFileRef = useRef<TarReader | null>(null);

  useEffect(() => {
    const initialize = async (file: ArrayBuffer) => {
      const tarReader = new TarReader();
      await tarReader.readFile(file);
      currentTarFileRef.current = tarReader;
      const fileTree = await createTemplateVersionFileTree(tarReader);
      setFileTree(fileTree);
    };

    if (fileQuery.data) {
      initialize(fileQuery.data).catch(() => {
        console.error("Error on initializing the editor");
      });
    }
  }, [fileQuery.data, sendEvent]);

  return (
    <>
      <Helmet>
        <title>{pageTitle(`${templateName} · Template Editor`)}</title>
      </Helmet>

      {templateQuery.data && templateVersionQuery.data && fileTree && (
        <TemplateVersionEditor
          template={templateQuery.data}
          templateVersion={
            editorState.context.version || templateVersionQuery.data
          }
          isBuildingNewVersion={Boolean(editorState.context.version)}
          defaultFileTree={fileTree}
          onPreview={async (newFileTree) => {
            if (!currentTarFileRef.current) {
              return;
            }

            const newVersionFile = await generateVersionFiles(
              currentTarFileRef.current,
              newFileTree,
            );
            const newVersionUpload = await uploadFileMutation.mutateAsync(
              newVersionFile,
            );
            sendEvent({
              type: "CREATE_VERSION",
              fileId: newVersionUpload.hash,
            });
          }}
          onPublish={() => {
            sendEvent({
              type: "PUBLISH",
            });
          }}
          onCancelPublish={() => {
            sendEvent({
              type: "CANCEL_PUBLISH",
            });
          }}
          onConfirmPublish={(data) => {
            sendEvent({
              type: "CONFIRM_PUBLISH",
              ...data,
            });
          }}
          isAskingPublishParameters={editorState.matches(
            "askPublishParameters",
          )}
          isPublishing={editorState.matches("publishingVersion")}
          publishingError={editorState.context.publishingError}
          publishedVersion={editorState.context.lastSuccessfulPublishedVersion}
          onCreateWorkspace={() => {
            const params = new URLSearchParams();
            const publishedVersion =
              editorState.context.lastSuccessfulPublishedVersion;
            if (publishedVersion) {
              params.set("version", publishedVersion.id);
            }
            navigate(
              `/templates/${templateName}/workspace?${params.toString()}`,
            );
          }}
          disablePreview={editorState.hasTag("loading")}
          disableUpdate={
            editorState.hasTag("loading") ||
            editorState.context.version?.job.status !== "succeeded"
          }
          resources={editorState.context.resources}
          buildLogs={editorState.context.buildLogs}
          isPromptingMissingVariables={editorState.matches("promptVariables")}
          missingVariables={editorState.context.missingVariables}
          onSubmitMissingVariableValues={(values) => {
            sendEvent({
              type: "SET_MISSING_VARIABLE_VALUES",
              values,
              fileId: uploadFileMutation.data!.hash,
            });
          }}
          onCancelSubmitMissingVariableValues={() => {
            sendEvent({
              type: "CANCEL_MISSING_VARIABLE_VALUES",
            });
          }}
        />
      )}
    </>
  );
};

const generateVersionFiles = async (
  tarReader: TarReader,
  fileTree: FileTree,
) => {
  const tar = new TarWriter();

  // Add previous non editable files
  for (const file of tarReader.fileInfo) {
    if (!isAllowedFile(file.name)) {
      if (file.type === "5") {
        tar.addFolder(file.name, {
          mode: file.mode, // https://github.com/beatgammit/tar-js/blob/master/lib/tar.js#L42
          mtime: file.mtime,
          user: file.user,
          group: file.group,
        });
      } else {
        tar.addFile(file.name, tarReader.getTextFile(file.name) as string, {
          mode: file.mode, // https://github.com/beatgammit/tar-js/blob/master/lib/tar.js#L42
          mtime: file.mtime,
          user: file.user,
          group: file.group,
        });
      }
    }
  }
  // Add the editable files
  traverse(fileTree, (content, _filename, fullPath) => {
    // When a file is deleted. Don't add it to the tar.
    if (content === undefined) {
      return;
    }

    if (typeof content === "string") {
      tar.addFile(fullPath, content);
      return;
    }

    tar.addFolder(fullPath);
  });
  const blob = (await tar.write()) as Blob;
  return new File([blob], "template.tar");
};

export default TemplateVersionEditorPage;
