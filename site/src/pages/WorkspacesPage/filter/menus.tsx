import { API } from "api/api";
import type { WorkspaceStatus } from "api/typesGenerated";
import { FilterMenu } from "components/Filter/filter";
import {
  useFilterMenu,
  type UseFilterMenuOptions,
} from "components/Filter/menu";
import type { SelectFilterOption } from "components/SelectFilter/SelectFilter";
import { StatusIndicator } from "components/StatusIndicator/StatusIndicator";
import { TemplateAvatar } from "components/TemplateAvatar/TemplateAvatar";
import { getDisplayWorkspaceStatus } from "utils/workspace";

export const useTemplateFilterMenu = ({
  value,
  onChange,
  organizationId,
}: { organizationId: string } & Pick<
  UseFilterMenuOptions<SelectFilterOption>,
  "value" | "onChange"
>) => {
  return useFilterMenu({
    onChange,
    value,
    id: "template",
    getSelectedOption: async () => {
      // Show all templates including deprecated
      const templates = await API.getTemplates(organizationId);
      const template = templates.find((template) => template.name === value);
      if (template) {
        return {
          label:
            template.display_name !== ""
              ? template.display_name
              : template.name,
          value: template.name,
          startIcon: <TemplateAvatar size="xs" template={template} />,
        };
      }
      return null;
    },
    getOptions: async (query) => {
      // Show all templates including deprecated
      const templates = await API.getTemplates(organizationId);
      const filteredTemplates = templates.filter(
        (template) =>
          template.name.toLowerCase().includes(query.toLowerCase()) ||
          template.display_name.toLowerCase().includes(query.toLowerCase()),
      );
      return filteredTemplates.map((template) => ({
        label:
          template.display_name !== "" ? template.display_name : template.name,
        value: template.name,
        startIcon: <TemplateAvatar size="xs" template={template} />,
      }));
    },
  });
};

export type TemplateFilterMenu = ReturnType<typeof useTemplateFilterMenu>;

export const TemplateMenu = (menu: TemplateFilterMenu) => {
  return (
    <FilterMenu
      placeholder="All templates"
      options={menu.searchOptions}
      onSelect={menu.selectOption}
      searchAriaLabel="Search template"
      searchPlaceholder="Search template..."
      selectedOption={menu.selectedOption ?? undefined}
      search={menu.query}
      onSearchChange={menu.setQuery}
    />
  );
};

/** Status Filter Menu */

export const useStatusFilterMenu = ({
  value,
  onChange,
}: Pick<UseFilterMenuOptions<SelectFilterOption>, "value" | "onChange">) => {
  const statusesToFilter: WorkspaceStatus[] = [
    "running",
    "stopped",
    "failed",
    "pending",
  ];
  const statusOptions = statusesToFilter.map((status) => {
    const display = getDisplayWorkspaceStatus(status);
    return {
      label: display.text,
      value: status,
      startIcon: <StatusIndicator color={display.type ?? "warning"} />,
    };
  });
  return useFilterMenu({
    onChange,
    value,
    id: "status",
    getSelectedOption: async () =>
      statusOptions.find((option) => option.value === value) ?? null,
    getOptions: async () => statusOptions,
  });
};

export type StatusFilterMenu = ReturnType<typeof useStatusFilterMenu>;

export const StatusMenu = (menu: StatusFilterMenu) => {
  return (
    <FilterMenu
      placeholder="All statuses"
      options={menu.searchOptions}
      selectedOption={menu.selectedOption ?? undefined}
      onSelect={menu.selectOption}
    />
  );
};
