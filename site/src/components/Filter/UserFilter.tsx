import type { FC } from "react";
import { API } from "api/api";
import type { SelectFilterOption } from "components/SelectFilter/SelectFilter";
import { UserAvatar } from "components/UserAvatar/UserAvatar";
import { useAuthenticated } from "contexts/auth/RequireAuth";
import { FilterMenu } from "./filter";
import { type UseFilterMenuOptions, useFilterMenu } from "./menu";

export const useUserFilterMenu = ({
  value,
  onChange,
  enabled,
}: Pick<
  UseFilterMenuOptions<SelectFilterOption>,
  "value" | "onChange" | "enabled"
>) => {
  const { user: me } = useAuthenticated();

  const addMeAsFirstOption = (options: SelectFilterOption[]) => {
    options = options.filter((option) => option.value !== me.username);
    return [
      {
        label: me.username,
        value: me.username,
        startIcon: (
          <UserAvatar
            username={me.username}
            avatarURL={me.avatar_url}
            size="xs"
          />
        ),
      },
      ...options,
    ];
  };

  return useFilterMenu({
    onChange,
    enabled,
    value,
    id: "owner",
    getSelectedOption: async () => {
      if (value === "me") {
        return {
          label: me.username,
          value: me.username,
          startIcon: (
            <UserAvatar
              username={me.username}
              avatarURL={me.avatar_url}
              size="xs"
            />
          ),
        };
      }

      const usersRes = await API.getUsers({ q: value, limit: 1 });
      const firstUser = usersRes.users.at(0);
      if (firstUser && firstUser.username === value) {
        return {
          label: firstUser.username,
          value: firstUser.username,
          startIcon: (
            <UserAvatar
              username={firstUser.username}
              avatarURL={firstUser.avatar_url}
              size="xs"
            />
          ),
        };
      }
      return null;
    },
    getOptions: async (query) => {
      const usersRes = await API.getUsers({ q: query, limit: 25 });
      let options: SelectFilterOption[] = usersRes.users.map((user) => ({
        label: user.username,
        value: user.username,
        startIcon: (
          <UserAvatar
            username={user.username}
            avatarURL={user.avatar_url}
            size="xs"
          />
        ),
      }));
      options = addMeAsFirstOption(options);
      return options;
    },
  });
};

export type UserFilterMenu = ReturnType<typeof useUserFilterMenu>;

interface UserMenuProps {
  menu: UserFilterMenu;
}

export const UserMenu: FC<UserMenuProps> = ({ menu }) => {
  return (
    <FilterMenu
      options={menu.searchOptions}
      onSelect={menu.selectOption}
      placeholder="All users"
      selectedOption={menu.selectedOption ?? undefined}
      search={menu.query}
      onSearchChange={menu.setQuery}
    />
  );
};
