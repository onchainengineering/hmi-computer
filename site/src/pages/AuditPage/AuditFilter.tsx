import { AuditActions, ResourceTypes } from "api/typesGenerated"
import { UserFilterMenu, UserMenu } from "components/Filter/UserFilter"
import {
  Filter,
  FilterMenu,
  MenuSkeleton,
  OptionItem,
  SearchFieldSkeleton,
  useFilter,
} from "components/Filter/filter"
import { UseFilterMenuOptions, useFilterMenu } from "components/Filter/menu"
import { BaseOption } from "components/Filter/options"
import capitalize from "lodash/capitalize"

export const AuditFilter = ({
  filter,
  error,
  menus,
}: {
  filter: ReturnType<typeof useFilter>
  error?: unknown
  menus: {
    user: UserFilterMenu
    action: ActionFilterMenu
    resourceType: ResourceTypeFilterMenu
  }
}) => {
  return (
    <Filter
      isLoading={menus.user.isInitializing}
      filter={filter}
      error={error}
      options={
        <>
          <ResourceTypeMenu {...menus.resourceType} />
          <ActionMenu {...menus.action} />
          <UserMenu {...menus.user} />
        </>
      }
      skeleton={
        <>
          <SearchFieldSkeleton />
          <MenuSkeleton />
          <MenuSkeleton />
          <MenuSkeleton />
        </>
      }
    />
  )
}

export const useActionFilterMenu = ({
  value,
  onChange,
}: Pick<UseFilterMenuOptions<BaseOption>, "value" | "onChange">) => {
  const actionOptions: BaseOption[] = AuditActions.map((action) => ({
    value: action,
    label: capitalize(action),
  }))
  return useFilterMenu({
    onChange,
    value,
    id: "status",
    getSelectedOption: async () =>
      actionOptions.find((option) => option.value === value) ?? null,
    getOptions: async () => actionOptions,
  })
}

export type ActionFilterMenu = ReturnType<typeof useActionFilterMenu>

const ActionMenu = (menu: ActionFilterMenu) => {
  return (
    <FilterMenu
      id="action-menu"
      menu={menu}
      label={
        menu.selectedOption ? (
          <OptionItem option={menu.selectedOption} />
        ) : (
          "All actions"
        )
      }
    >
      {(itemProps) => <OptionItem {...itemProps} />}
    </FilterMenu>
  )
}

export const useResourceTypeFilterMenu = ({
  value,
  onChange,
}: Pick<UseFilterMenuOptions<BaseOption>, "value" | "onChange">) => {
  const actionOptions: BaseOption[] = ResourceTypes.map((type) => {
    let label = capitalize(type)

    if (type === "api_key") {
      label = "API Key"
    }

    if (type === "git_ssh_key") {
      label = "Git SSH Key"
    }

    if (type === "template_version") {
      label = "Template Version"
    }

    if (type === "workspace_build") {
      label = "Workspace Build"
    }

    return {
      value: type,
      label,
    }
  })
  return useFilterMenu({
    onChange,
    value,
    id: "resourceType",
    getSelectedOption: async () =>
      actionOptions.find((option) => option.value === value) ?? null,
    getOptions: async () => actionOptions,
  })
}

export type ResourceTypeFilterMenu = ReturnType<
  typeof useResourceTypeFilterMenu
>

const ResourceTypeMenu = (menu: ResourceTypeFilterMenu) => {
  return (
    <FilterMenu
      id="resource-type-menu"
      menu={menu}
      label={
        menu.selectedOption ? (
          <OptionItem option={menu.selectedOption} />
        ) : (
          "All resource types"
        )
      }
    >
      {(itemProps) => <OptionItem {...itemProps} />}
    </FilterMenu>
  )
}
