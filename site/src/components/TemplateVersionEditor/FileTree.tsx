import { makeStyles } from "@material-ui/core/styles"
import ChevronRightIcon from "@material-ui/icons/ChevronRight"
import ExpandMoreIcon from "@material-ui/icons/ExpandMore"
import TreeView from "@material-ui/lab/TreeView"
import TreeItem from "@material-ui/lab/TreeItem"

import { FC, useMemo } from "react"
import { TemplateVersionFiles } from "util/templateVersion"

export interface File {
  path: string
  content?: string
  children: Record<string, File>
}

export const FileTree: FC<{
  onSelect: (file: File) => void
  files: TemplateVersionFiles
  activeFile?: File
}> = ({ activeFile, files, onSelect }) => {
  const styles = useStyles()
  const fileTree = useMemo<Record<string, File>>(() => {
    const paths = Object.keys(files)
    const roots: Record<string, File> = {}
    paths.forEach((path) => {
      const pathParts = path.split("/")
      const firstPart = pathParts.shift()
      if (!firstPart) {
        // Not possible!
        return
      }
      let activeFile = roots[firstPart]
      if (!activeFile) {
        activeFile = {
          path: firstPart,
          children: {},
        }
        roots[firstPart] = activeFile
      }
      while (pathParts.length > 0) {
        const pathPart = pathParts.shift()
        if (!pathPart) {
          continue
        }
        if (!activeFile.children[pathPart]) {
          activeFile.children[pathPart] = {
            path: activeFile.path + "/" + pathPart,
            children: {},
          }
        }
        activeFile = activeFile.children[pathPart]
      }
      activeFile.content = files[path]
      activeFile.path = path
    })
    return roots
  }, [files])

  const buildTreeItems = (name: string, file: File): JSX.Element => {
    let icon: JSX.Element | null = null
    if (file.path.endsWith(".tf")) {
      icon = <FileTypeTerraform />
    }
    if (file.path.endsWith(".md")) {
      icon = <FileTypeMarkdown />
    }
    if (file.path.endsWith("Dockerfile")) {
      icon = <FileTypeDockerfile />
    }

    return (
      <TreeItem
        nodeId={file.path}
        key={file.path}
        label={name}
        className={`${styles.fileTreeItem} ${
          file.path === activeFile?.path ? "active" : ""
        }`}
        onClick={() => {
          if (file.content) {
            onSelect(file)
          }
        }}
        icon={icon}
      >
        {Object.entries(file.children || {}).map(([name, file]) => {
          return buildTreeItems(name, file)
        })}
      </TreeItem>
    )
  }

  return (
    <TreeView
      defaultCollapseIcon={<ExpandMoreIcon />}
      defaultExpandIcon={<ChevronRightIcon />}
      aria-label="Files"
      className={styles.fileTree}
    >
      {Object.entries(fileTree).map(([name, file]) => {
        return buildTreeItems(name, file)
      })}
    </TreeView>
  )
}

const useStyles = makeStyles((theme) => ({
  fileTree: {},
  fileTreeItem: {
    overflow: "hidden",
    userSelect: "none",

    "&:focus": {
      "& > .MuiTreeItem-content": {
        background: "rgba(255, 255, 255, 0.1)",
      },
    },
    "& > .MuiTreeItem-content:hover": {
      background: theme.palette.background.paperLight,
      color: theme.palette.text.primary,
    },

    "& > .MuiTreeItem-content": {
      padding: "1px 16px",
      color: theme.palette.text.secondary,

      "& svg": {
        width: 16,
        height: 16,
      },

      "& > .MuiTreeItem-label": {
        marginLeft: 4,
        fontSize: 14,
        color: "inherit",
      },
    },

    "&.active": {
      background: theme.palette.background.paper,

      "& > .MuiTreeItem-content": {
        color: theme.palette.text.primary,
      },
    },
  },
  editor: {
    flex: 1,
  },
  preview: {},
}))

const FileTypeTerraform = () => (
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32" fill="#813cf3">
    <title>file_type_terraform</title>
    <polygon points="12.042 6.858 20.071 11.448 20.071 20.462 12.042 15.868 12.042 6.858 12.042 6.858" />
    <polygon points="20.5 20.415 28.459 15.84 28.459 6.887 20.5 11.429 20.5 20.415 20.5 20.415" />
    <polygon points="3.541 11.01 11.571 15.599 11.571 6.59 3.541 2 3.541 11.01 3.541 11.01" />
    <polygon points="12.042 25.41 20.071 30 20.071 20.957 12.042 16.368 12.042 25.41 12.042 25.41" />
  </svg>
)

const FileTypeMarkdown = () => (
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32" fill="#755838">
    <rect
      x="2.5"
      y="7.955"
      width="27"
      height="16.091"
      style={{
        fill: "none",
        stroke: "#755838",
      }}
    />
    <polygon points="5.909 20.636 5.909 11.364 8.636 11.364 11.364 14.773 14.091 11.364 16.818 11.364 16.818 20.636 14.091 20.636 14.091 15.318 11.364 18.727 8.636 15.318 8.636 20.636 5.909 20.636" />
    <polygon points="22.955 20.636 18.864 16.136 21.591 16.136 21.591 11.364 24.318 11.364 24.318 16.136 27.045 16.136 22.955 20.636" />
  </svg>
)

const FileTypeDockerfile = () => (
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32" fill="#0db7ed">
    <path d="M16,2A14,14,0,1,0,30,16,14,14,0,0,0,16,2Zm0,26A12,12,0,1,1,28,16,12,12,0,0,1,16,28Z" />
    <path d="M16,6a10,10,0,0,0-9.9,9.1H6.1A8,8,0,0,1,16,8a8,8,0,0,1,8,8,8,8,0,0,1-8,8,8,8,0,0,1-7.9-6.1H6.1A10,10,0,0,0,16,26a10,10,0,0,0,0-20Z" />
    <path d="M16,10a6,6,0,0,0-6,6,6,6,0,0,0,6,6,6,6,0,0,0,6-6A6,6,0,0,0,16,10Zm0,10a4,4,0,0,1-4-4,4,4,0,0,1,4-4,4,4,0,0,1,4,4A4,4,0,0,1,16,20Z" />
  </svg>
)
