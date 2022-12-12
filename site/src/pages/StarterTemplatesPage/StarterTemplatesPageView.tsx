import { makeStyles } from "@material-ui/core/styles"
import { AlertBanner } from "components/AlertBanner/AlertBanner"
import { Maybe } from "components/Conditionals/Maybe"
import { Loader } from "components/Loader/Loader"
import { Margins } from "components/Margins/Margins"
import {
  PageHeader,
  PageHeaderSubtitle,
  PageHeaderTitle,
} from "components/PageHeader/PageHeader"
import { Stack } from "components/Stack/Stack"
import { FC } from "react"
import { useTranslation } from "react-i18next"
import { Link, useSearchParams } from "react-router-dom"
import { combineClasses } from "util/combineClasses"
import { StarterTemplatesContext } from "xServices/starterTemplates/starterTemplatesXService"

const getTagLabel = (tag: string, t: (key: string) => string) => {
  const labelByTag: Record<string, string> = {
    all: t("tags.all"),
    digitalocean: t("tags.digitalocean"),
    aws: t("tags.aws"),
    google: t("tags.google"),
  }
  // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- this can be undefined
  return labelByTag[tag] ?? tag
}

const selectTags = ({ starterTemplatesByTag }: StarterTemplatesContext) => {
  return starterTemplatesByTag
    ? Object.keys(starterTemplatesByTag).sort((a, b) => a.localeCompare(b))
    : undefined
}
export interface StarterTemplatesPageViewProps {
  context: StarterTemplatesContext
}

export const StarterTemplatesPageView: FC<StarterTemplatesPageViewProps> = ({
  context,
}) => {
  const { t } = useTranslation("starterTemplatesPage")
  const [urlParams] = useSearchParams()
  const styles = useStyles()
  const { starterTemplatesByTag } = context
  const tags = selectTags(context)
  const activeTag = urlParams.get("tag") ?? "all"
  const visibleTemplates = starterTemplatesByTag
    ? starterTemplatesByTag[activeTag]
    : undefined

  return (
    <Margins>
      <PageHeader>
        <PageHeaderTitle>{t("title")}</PageHeaderTitle>
        <PageHeaderSubtitle>{t("subtitle")}</PageHeaderSubtitle>
      </PageHeader>

      <Maybe condition={Boolean(context.error)}>
        <AlertBanner error={context.error} severity="error" />
      </Maybe>

      <Maybe condition={Boolean(!starterTemplatesByTag)}>
        <Loader />
      </Maybe>

      <Stack direction="row" spacing={4}>
        {starterTemplatesByTag && tags && (
          <Stack className={styles.filter}>
            <span className={styles.filterCaption}>{t("filterCaption")}</span>
            {tags.map((tag) => (
              <Link
                key={tag}
                to={`?tag=${tag}`}
                className={combineClasses({
                  [styles.tagLink]: true,
                  [styles.tagLinkActive]: tag === activeTag,
                })}
              >
                {getTagLabel(tag, t)} ({starterTemplatesByTag[tag].length})
              </Link>
            ))}
          </Stack>
        )}

        <div className={styles.templates}>
          {visibleTemplates &&
            visibleTemplates.map((example) => (
              <Link
                to={example.id}
                className={styles.template}
                key={example.id}
              >
                <div className={styles.templateIcon}>
                  <img src={example.icon} alt="" />
                </div>
                <div className={styles.templateInfo}>
                  <span className={styles.templateName}>{example.name}</span>
                  <span className={styles.templateDescription}>
                    {example.description}
                  </span>
                </div>
              </Link>
            ))}
        </div>
      </Stack>
    </Margins>
  )
}

const useStyles = makeStyles((theme) => ({
  filter: {
    width: theme.spacing(26),
    flexShrink: 0,
  },

  filterCaption: {
    textTransform: "uppercase",
    fontWeight: 600,
    fontSize: 12,
    color: theme.palette.text.secondary,
    letterSpacing: "0.1em",
  },

  tagLink: {
    color: theme.palette.text.secondary,
    textDecoration: "none",
    fontSize: 14,
    textTransform: "capitalize",

    "&:hover": {
      color: theme.palette.text.primary,
    },
  },

  tagLinkActive: {
    color: theme.palette.text.primary,
    fontWeight: 600,
  },

  templates: {
    flex: "1",
    display: "grid",
    gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
    gap: theme.spacing(2),
    gridAutoRows: "min-content",
  },

  template: {
    border: `1px solid ${theme.palette.divider}`,
    borderRadius: theme.shape.borderRadius,
    background: theme.palette.background.paper,
    textDecoration: "none",
    color: "inherit",
    display: "flex",
    alignItems: "center",
    height: "fit-content",

    "&:hover": {
      backgroundColor: theme.palette.background.paperLight,
    },
  },

  templateIcon: {
    width: theme.spacing(12),
    height: theme.spacing(12),
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    flexShrink: 0,

    "& img": {
      height: theme.spacing(4),
    },
  },

  templateInfo: {
    padding: theme.spacing(2, 2, 2, 0),
    display: "flex",
    flexDirection: "column",
    gap: theme.spacing(0.5),
    overflow: "hidden",
  },

  templateName: {
    fontSize: theme.spacing(2),
  },

  templateDescription: {
    fontSize: theme.spacing(1.75),
    color: theme.palette.text.secondary,
    textOverflow: "ellipsis",
    width: "100%",
    overflow: "hidden",
    whiteSpace: "nowrap",
  },
}))
