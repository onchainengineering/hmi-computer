import { type PropsWithChildren, type ReactNode, useState } from "react";
import { type Template } from "api/typesGenerated";
import { type UseQueryResult } from "react-query";
import {
  Link as RouterLink,
  LinkProps as RouterLinkProps,
} from "react-router-dom";
import Box from "@mui/system/Box";
import Button from "@mui/material/Button";
import Link from "@mui/material/Link";
import AddIcon from "@mui/icons-material/AddOutlined";
import OpenIcon from "@mui/icons-material/OpenInNewOutlined";
import { Loader } from "components/Loader/Loader";
import { OverflowY } from "components/OverflowY/OverflowY";
import { EmptyState } from "components/EmptyState/EmptyState";
import { Avatar } from "components/Avatar/Avatar";
import { SearchBox } from "./WorkspacesSearchBox";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "components/Popover/Popover";

const ICON_SIZE = 18;

type TemplatesQuery = UseQueryResult<Template[]>;

type WorkspacesButtonProps = PropsWithChildren<{
  templatesFetchStatus: TemplatesQuery["status"];
  templates: TemplatesQuery["data"];
}>;

export function WorkspacesButton({
  children,
  templatesFetchStatus,
  templates,
}: WorkspacesButtonProps) {
  // Dataset should always be small enough that client-side filtering should be
  // good enough. Can swap out down the line if it becomes an issue
  const [searchTerm, setSearchTerm] = useState("");
  const processed = sortTemplatesByUsersDesc(templates ?? [], searchTerm);

  let emptyState: ReactNode = undefined;
  if (templates?.length === 0) {
    emptyState = (
      <EmptyState
        message="No templates yet"
        cta={
          <Link to="/templates" component={RouterLink}>
            Create one now.
          </Link>
        }
      />
    );
  } else if (processed.length === 0) {
    emptyState = <EmptyState message="No templates match your text" />;
  }

  return (
    <Popover>
      <PopoverTrigger>
        <Button startIcon={<AddIcon />} variant="contained">
          {children}
        </Button>
      </PopoverTrigger>
      <PopoverContent horizontal="right">
        <SearchBox
          value={searchTerm}
          onValueChange={(newValue) => setSearchTerm(newValue)}
          placeholder="Type/select a workspace template"
          label="Template select for workspace"
          sx={{ flexShrink: 0, columnGap: 1.5 }}
        />

        <OverflowY
          maxHeight={380}
          sx={{
            display: "flex",
            flexDirection: "column",
            paddingY: 1,
          }}
        >
          {templatesFetchStatus === "loading" ? (
            <Loader size={14} />
          ) : (
            <>
              {processed.map((template) => (
                <WorkspaceResultsRow key={template.id} template={template} />
              ))}

              {emptyState}
            </>
          )}
        </OverflowY>

        <Box
          css={(theme) => ({
            padding: "8px 0",
            borderTop: `1px solid ${theme.deprecated.palette.divider}`,
          })}
        >
          <PopoverLink
            to="/templates"
            css={(theme) => ({
              display: "flex",
              alignItems: "center",
              columnGap: 12,

              color: theme.deprecated.palette.primary.main,
            })}
          >
            <OpenIcon css={{ width: 14, height: 14 }} />
            <span>See all templates</span>
          </PopoverLink>
        </Box>
      </PopoverContent>
    </Popover>
  );
}

function WorkspaceResultsRow({ template }: { template: Template }) {
  return (
    <PopoverLink
      to={`/templates/${template.name}/workspace`}
      css={{
        display: "flex",
        gap: 12,
        alignItems: "center",
      }}
    >
      <Avatar
        src={template.icon}
        fitImage
        alt={template.display_name || "Coder template"}
        sx={{
          width: `${ICON_SIZE}px`,
          height: `${ICON_SIZE}px`,
          fontSize: `${ICON_SIZE * 0.5}px`,
          fontWeight: 700,
        }}
      >
        {template.display_name || "-"}
      </Avatar>

      <Box
        css={(theme) => ({
          color: theme.deprecated.palette.text.primary,
          display: "flex",
          flexDirection: "column",
          lineHeight: "140%",
          fontSize: 14,
          overflow: "hidden",
        })}
      >
        <span css={{ whiteSpace: "nowrap", textOverflow: "ellipsis" }}>
          {template.display_name || template.name || "[Unnamed]"}
        </span>
        <span
          css={(theme) => ({
            fontSize: 13,
            color: theme.deprecated.palette.text.secondary,
          })}
        >
          {/*
           * There are some templates that have -1 as their user count –
           * basically functioning like a null value in JS. Can safely just
           * treat them as if they were 0.
           */}
          {template.active_user_count <= 0 ? "No" : template.active_user_count}{" "}
          developer
          {template.active_user_count === 1 ? "" : "s"}
        </span>
      </Box>
    </PopoverLink>
  );
}

function PopoverLink(props: RouterLinkProps) {
  return (
    <RouterLink
      {...props}
      css={(theme) => ({
        color: theme.deprecated.palette.text.primary,
        padding: "8px 16px",
        fontSize: 14,
        outline: "none",
        textDecoration: "none",
        "&:focus": {
          backgroundColor: theme.deprecated.palette.action.focus,
        },
        "&:hover": {
          textDecoration: "none",
          backgroundColor: theme.deprecated.palette.action.hover,
        },
      })}
    />
  );
}

function sortTemplatesByUsersDesc(
  templates: readonly Template[],
  searchTerm: string,
) {
  const allWhitespace = /^\s+$/.test(searchTerm);
  if (allWhitespace) {
    return templates;
  }

  const termMatcher = new RegExp(searchTerm.replaceAll(/[^\w]/g, "."), "i");
  return templates
    .filter(
      (template) =>
        termMatcher.test(template.display_name) ||
        termMatcher.test(template.name),
    )
    .sort((t1, t2) => t2.active_user_count - t1.active_user_count)
    .slice(0, 10);
}
