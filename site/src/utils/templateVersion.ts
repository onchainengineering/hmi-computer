import set from "lodash/set";
import { FileTree } from "./filetree";
import { TarFileTypeCodes, TarReader } from "./tar";

// Content by filename
export type TemplateVersionFiles = Record<string, string>;

export const getTemplateVersionFiles = async (
  tarFile: ArrayBuffer,
): Promise<TemplateVersionFiles> => {
  const files: TemplateVersionFiles = {};
  const tarReader = new TarReader();
  await tarReader.readFile(tarFile);
  for (const file of tarReader.fileInfo) {
    files[file.name] = tarReader.getTextFile(file.name) as string;
  }
  return files;
};

export const createTemplateVersionFileTree = async (
  tarReader: TarReader,
): Promise<FileTree> => {
  let fileTree: FileTree = {};
  for (const file of tarReader.fileInfo) {
    fileTree = set(
      fileTree,
      file.name.split("/"),
      file.type === TarFileTypeCodes.Dir
        ? {}
        : (tarReader.getTextFile(file.name) as string),
    );
  }
  return fileTree;
};
