import { makeStyles } from "@material-ui/core/styles"
import { MarkdownIcon } from "components/Icons/MarkdownIcon"
import { TerraformIcon } from "components/Icons/TerraformIcon"
import { Loader } from "components/Loader/Loader"
import { Margins } from "components/Margins/Margins"
import {
  PageHeader,
  PageHeaderCaption,
  PageHeaderTitle,
} from "components/PageHeader/PageHeader"
import { Stack } from "components/Stack/Stack"
import { Stats, StatsItem } from "components/Stats/Stats"
import { SyntaxHighlighter } from "components/SyntaxHighlighter/SyntaxHighlighter"
import { UseTabResult } from "hooks/useTab"
import { FC } from "react"
import { Link } from "react-router-dom"
import { combineClasses } from "util/combineClasses"
import { createDayString } from "util/createDayString"
import { TemplateVersionMachineContext } from "xServices/templateVersion/templateVersionXService"

export interface TemplateVersionPageViewProps {
  /**
   * Used to display the version name before loading the version in the API
   */
  versionName: string
  templateName: string
  tab: UseTabResult
  context: TemplateVersionMachineContext
}

export const TemplateVersionPageView: FC<TemplateVersionPageViewProps> = ({
  context,
  tab,
  versionName,
  templateName,
}) => {
  const styles = useStyles()
  const { files, error, version } = context

  return (
    <>
      <Margins>
        <PageHeader>
          <PageHeaderCaption>Versions</PageHeaderCaption>
          <PageHeaderTitle>{versionName}</PageHeaderTitle>
        </PageHeader>
      </Margins>

      {!files && !error && <Loader />}

      {version && files && (
        <Stack spacing={4}>
          <Margins>
            <Stats>
              <StatsItem
                label="Template"
                value={
                  <Link to={`/templates/${templateName}`}>{templateName}</Link>
                }
              />
              <StatsItem
                label="Created by"
                value={version.created_by.username}
              />
              <StatsItem
                label="Created"
                value={createDayString(version.created_at)}
              />
            </Stats>
          </Margins>

          <Margins>
            <div className={styles.files}>
              <div className={styles.tabs}>
                {Object.keys(files).map((filename, index) => {
                  const tabValue = index.toString()

                  return (
                    <button
                      className={combineClasses({
                        [styles.tab]: true,
                        [styles.tabActive]: tabValue === tab.value,
                      })}
                      onClick={() => {
                        tab.set(tabValue)
                      }}
                      key={filename}
                    >
                      {filename.endsWith("tf") ? (
                        <TerraformIcon />
                      ) : (
                        <MarkdownIcon />
                      )}
                      {filename}
                    </button>
                  )
                })}
              </div>

              <SyntaxHighlighter
                showLineNumbers
                className={styles.prism}
                language={
                  Object.keys(files)[Number(tab.value)].endsWith("tf")
                    ? "hcl"
                    : "markdown"
                }
              >
                {Object.values(files)[Number(tab.value)]}
              </SyntaxHighlighter>
            </div>
          </Margins>
        </Stack>
      )}
    </>
  )
}

const useStyles = makeStyles((theme) => ({
  tabsWrapper: {
    borderBottom: `1px solid ${theme.palette.divider}`,
  },

  tabs: {
    display: "flex",
    alignItems: "baseline",
    borderBottom: `1px solid ${theme.palette.divider}`,
  },

  tab: {
    background: "transparent",
    border: 0,
    padding: theme.spacing(0, 3),
    display: "flex",
    alignItems: "center",
    height: theme.spacing(6),
    opacity: 0.75,
    cursor: "pointer",
    gap: theme.spacing(0.5),
    position: "relative",

    "& svg": {
      width: 22,
      maxHeight: 16,
    },

    "&:hover": {
      backgroundColor: theme.palette.action.hover,
    },
  },

  tabActive: {
    opacity: 1,
    fontWeight: 600,

    "&:after": {
      content: '""',
      display: "block",
      height: 1,
      width: "100%",
      bottom: 0,
      left: 0,
      backgroundColor: theme.palette.secondary.dark,
      position: "absolute",
    },
  },

  codeWrapper: {
    background: theme.palette.background.paperLight,
  },

  files: {
    borderRadius: theme.shape.borderRadius,
    border: `1px solid ${theme.palette.divider}`,
  },

  prism: {
    borderRadius: 0,
  },
}))

export default TemplateVersionPageView
