import { FC } from "react"
import Box from "@mui/material/Box"
import { UserAvatar } from "components/UserAvatar/UserAvatar"
import { Avatar, AvatarProps } from "components/Avatar/Avatar"
import { Palette, PaletteColor } from "@mui/material/styles"
import { UserFilterMenu, TemplateFilterMenu, StatusFilterMenu } from "./menus"
import { UserOption, TemplateOption, StatusOption } from "./options"
import {
  Filter,
  FilterMenu,
  FilterSearchMenu,
  MenuSkeleton,
  OptionItem,
  SearchFieldSkeleton,
  useFilter,
} from "components/Filter/filter"

export const WorkspacesFilter = ({
  filter,
  error,
  menus,
}: {
  filter: ReturnType<typeof useFilter>
  error?: unknown
  menus: {
    user?: UserFilterMenu
    template: TemplateFilterMenu
    status: StatusFilterMenu
  }
}) => {
  return (
    <Filter
      isLoading={menus.status.isInitializing}
      filter={filter}
      error={error}
      options={
        <>
          {menus.user && <UserMenu {...menus.user} />}
          <TemplateMenu {...menus.template} />
          <StatusMenu {...menus.status} />
        </>
      }
      skeleton={
        <>
          <SearchFieldSkeleton />
          {menus.user && <MenuSkeleton />}
          <MenuSkeleton />
          <MenuSkeleton />
        </>
      }
    />
  )
}

const UserMenu = (menu: UserFilterMenu) => {
  return (
    <FilterSearchMenu
      id="users-menu"
      menu={menu}
      label={
        menu.selectedOption ? (
          <UserOptionItem option={menu.selectedOption} />
        ) : (
          "All users"
        )
      }
    >
      {(itemProps) => <UserOptionItem {...itemProps} />}
    </FilterSearchMenu>
  )
}

const UserOptionItem = ({
  option,
  isSelected,
}: {
  option: UserOption
  isSelected?: boolean
}) => {
  return (
    <OptionItem
      option={option}
      isSelected={isSelected}
      left={
        <UserAvatar
          username={option.label}
          avatarURL={option.avatarUrl}
          sx={{ width: 16, height: 16, fontSize: 8 }}
        />
      }
    />
  )
}

const TemplateMenu = (menu: TemplateFilterMenu) => {
  return (
    <FilterSearchMenu
      id="templates-menu"
      menu={menu}
      label={
        menu.selectedOption ? (
          <TemplateOptionItem option={menu.selectedOption} />
        ) : (
          "All templates"
        )
      }
    >
      {(itemProps) => <TemplateOptionItem {...itemProps} />}
    </FilterSearchMenu>
  )
}

const TemplateOptionItem = ({
  option,
  isSelected,
}: {
  option: TemplateOption
  isSelected?: boolean
}) => {
  return (
    <OptionItem
      option={option}
      isSelected={isSelected}
      left={
        <TemplateAvatar
          templateName={option.label}
          icon={option.icon}
          sx={{ width: 14, height: 14, fontSize: 8 }}
        />
      }
    />
  )
}

const TemplateAvatar: FC<
  AvatarProps & { templateName: string; icon?: string }
> = ({ templateName, icon, ...avatarProps }) => {
  return icon ? (
    <Avatar src={icon} variant="square" fitImage {...avatarProps} />
  ) : (
    <Avatar {...avatarProps}>{templateName}</Avatar>
  )
}

const StatusMenu = (menu: StatusFilterMenu) => {
  return (
    <FilterMenu
      id="status-menu"
      menu={menu}
      label={
        menu.selectedOption ? (
          <StatusOptionItem option={menu.selectedOption} />
        ) : (
          "All statuses"
        )
      }
    >
      {(itemProps) => <StatusOptionItem {...itemProps} />}
    </FilterMenu>
  )
}

const StatusOptionItem = ({
  option,
  isSelected,
}: {
  option: StatusOption
  isSelected?: boolean
}) => {
  return (
    <OptionItem
      option={option}
      left={<StatusIndicator option={option} />}
      isSelected={isSelected}
    />
  )
}

const StatusIndicator: FC<{ option: StatusOption }> = ({ option }) => {
  return (
    <Box
      height={8}
      width={8}
      borderRadius={9999}
      sx={{
        backgroundColor: (theme) =>
          (theme.palette[option.color as keyof Palette] as PaletteColor).light,
      }}
    />
  )
}
