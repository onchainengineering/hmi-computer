import Link from "@material-ui/core/Link"
import { makeStyles } from "@material-ui/core/styles"
import Table from "@material-ui/core/Table"
import TableBody from "@material-ui/core/TableBody"
import TableCell from "@material-ui/core/TableCell"
import TableHead from "@material-ui/core/TableHead"
import TableRow from "@material-ui/core/TableRow"
import dayjs from "dayjs"
import relativeTime from "dayjs/plugin/relativeTime"
import { FC } from "react"
import * as TypesGen from "../../api/typesGenerated"
import { AvatarData } from "../../components/AvatarData/AvatarData"
import { CodeExample } from "../../components/CodeExample/CodeExample"
import { EmptyState } from "../../components/EmptyState/EmptyState"
import {
  HelpTooltip,
  HelpTooltipLink,
  HelpTooltipLinksGroup,
  HelpTooltipText,
  HelpTooltipTitle,
} from "../../components/HelpTooltip/HelpTooltip"
import { Margins } from "../../components/Margins/Margins"
import { PageHeader, PageHeaderTitle } from "../../components/PageHeader/PageHeader"
import { Stack } from "../../components/Stack/Stack"
import { TableLoader } from "../../components/TableLoader/TableLoader"

dayjs.extend(relativeTime)

export const Language = {
  developerCount: (ownerCount: number): string => {
    return `${ownerCount} developer${ownerCount !== 1 ? "s" : ""}`
  },
  nameLabel: "Name",
  usedByLabel: "Used by",
  lastUpdatedLabel: "Last updated",
  emptyViewNoPerms: "Contact your Coder administrator to create a template. You can share the code below.",
  emptyMessage: "Create your first template",
  emptyDescription: (
    <>
      To create a workspace you need to have a template. You can{" "}
      <Link target="_blank" href="https://github.com/coder/coder/blob/main/docs/templates.md">
        create one from scratch
      </Link>{" "}
      or use a built-in template using the following Coder CLI command:
    </>
  ),
}

const TemplateHelpTooltip: React.FC = () => {
  return (
    <HelpTooltip>
      <HelpTooltipTitle>What is template?</HelpTooltipTitle>
      <HelpTooltipText>
        With templates you can create a common configuration for your workspaces using Terraform. So, you and your team
        can use the same environment to deliver great software.
      </HelpTooltipText>
      <HelpTooltipLinksGroup>
        <HelpTooltipLink href="https://github.com/coder/coder/blob/main/docs/templates.md#manage-templates">
          Manage templates
        </HelpTooltipLink>
      </HelpTooltipLinksGroup>
    </HelpTooltip>
  )
}

export interface TemplatesPageViewProps {
  loading?: boolean
  canCreateTemplate?: boolean
  templates?: TypesGen.Template[]
}

export const TemplatesPageView: FC<TemplatesPageViewProps> = (props) => {
  const styles = useStyles()
  return (
    <Margins>
      <PageHeader>
        <PageHeaderTitle>
          <Stack spacing={1} direction="row" alignItems="center">
            Templates
            <TemplateHelpTooltip />
          </Stack>
        </PageHeaderTitle>
      </PageHeader>

      <Table>
        <TableHead>
          <TableRow>
            <TableCell>{Language.nameLabel}</TableCell>
            <TableCell>{Language.usedByLabel}</TableCell>
            <TableCell>{Language.lastUpdatedLabel}</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {props.loading && <TableLoader />}
          {!props.loading && !props.templates?.length && (
            <TableRow>
              <TableCell colSpan={999}>
                <EmptyState
                  message={Language.emptyMessage}
                  description={props.canCreateTemplate ? Language.emptyDescription : Language.emptyViewNoPerms}
                  descriptionClassName={styles.emptyDescription}
                  cta={<CodeExample code="coder template init" />}
                />
              </TableCell>
            </TableRow>
          )}
          {props.templates?.map((template) => (
            <TableRow key={template.id}>
              <TableCell>
                <AvatarData
                  title={template.name}
                  subtitle={template.description}
                  link={`/templates/${template.name}`}
                />
              </TableCell>

              <TableCell>{Language.developerCount(template.workspace_owner_count)}</TableCell>

              <TableCell data-chromatic="ignore">{dayjs().to(dayjs(template.updated_at))}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Margins>
  )
}

const useStyles = makeStyles((theme) => ({
  emptyDescription: {
    maxWidth: theme.spacing(62),
  },
}))
