import VpnKeyOutlined from "@mui/icons-material/VpnKeyOutlined";
import FingerprintOutlinedIcon from "@mui/icons-material/FingerprintOutlined";
import AccountIcon from "@mui/icons-material/Person";
import ScheduleIcon from "@mui/icons-material/EditCalendarOutlined";
import SecurityIcon from "@mui/icons-material/LockOutlined";
import type { User } from "api/typesGenerated";
import { UserAvatar } from "components/UserAvatar/UserAvatar";
import {
  Sidebar as BaseSidebar,
  SidebarHeader,
  SidebarNavItem,
} from "components/Sidebar/Sidebar";
import { GitIcon } from "components/Icons/GitIcon";

export const Sidebar: React.FC<{ user: User }> = ({ user }) => {
  return (
    <BaseSidebar>
      <SidebarHeader
        avatar={
          <UserAvatar username={user.username} avatarURL={user.avatar_url} />
        }
        title={user.username}
        subtitle={user.email}
      />
      <SidebarNavItem href="account" icon={AccountIcon}>
        Account
      </SidebarNavItem>
      <SidebarNavItem href="schedule" icon={ScheduleIcon}>
        Schedule
      </SidebarNavItem>
      <SidebarNavItem href="security" icon={SecurityIcon}>
        Security
      </SidebarNavItem>
      <SidebarNavItem href="ssh-keys" icon={FingerprintOutlinedIcon}>
        SSH Keys
      </SidebarNavItem>
      <SidebarNavItem href="external-auth" icon={GitIcon}>
        External Authentication
      </SidebarNavItem>
      <SidebarNavItem href="tokens" icon={VpnKeyOutlined}>
        Tokens
      </SidebarNavItem>
    </BaseSidebar>
  );
};
