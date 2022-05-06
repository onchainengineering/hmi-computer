import Box from "@material-ui/core/Box"
import Table from "@material-ui/core/Table"
import TableBody from "@material-ui/core/TableBody"
import TableCell from "@material-ui/core/TableCell"
import TableHead from "@material-ui/core/TableHead"
import TableRow from "@material-ui/core/TableRow"
import React from "react"
import { UserResponse } from "../../api/types"
import * as TypesGen from "../../api/typesGenerated"
import { EmptyState } from "../EmptyState/EmptyState"
import { TableHeaderRow } from "../TableHeaders/TableHeaders"
import { TableRowMenu } from "../TableRowMenu/TableRowMenu"
import { TableTitle } from "../TableTitle/TableTitle"
import { UserCell } from "../UserCell/UserCell"

export const Language = {
  pageTitle: "Users",
  usersTitle: "All users",
  emptyMessage: "No users found",
  usernameLabel: "User",
  suspendMenuItem: "Suspend",
  resetPasswordMenuItem: "Reset password",
  rolesLabel: "Roles",
}

export interface UsersTableProps {
  users: UserResponse[]
  onSuspendUser: (user: UserResponse) => void
  onResetUserPassword: (user: UserResponse) => void
  roles: TypesGen.Role[]
}

export const UsersTable: React.FC<UsersTableProps> = ({ users, roles, onSuspendUser, onResetUserPassword }) => {
  return (
    <Table>
      <TableHead>
        <TableTitle title={Language.usersTitle} />
        <TableHeaderRow>
          <TableCell size="small">{Language.usernameLabel}</TableCell>
          <TableCell size="small">{Language.rolesLabel}</TableCell>
          {/* 1% is a trick to make the table cell width fit the content */}
          <TableCell size="small" width="1%" />
        </TableHeaderRow>
      </TableHead>
      <TableBody>
        {users.map((u) => (
          <TableRow key={u.id}>
            <TableCell>
              <UserCell Avatar={{ username: u.username }} primaryText={u.username} caption={u.email} />{" "}
            </TableCell>
            <TableCell>{roles.map((r) => r.display_name)}</TableCell>
            <TableCell>
              <TableRowMenu
                data={u}
                menuItems={[
                  {
                    label: Language.suspendMenuItem,
                    onClick: onSuspendUser,
                  },
                  {
                    label: Language.resetPasswordMenuItem,
                    onClick: onResetUserPassword,
                  },
                ]}
              />
            </TableCell>
          </TableRow>
        ))}

        {users.length === 0 && (
          <TableRow>
            <TableCell colSpan={999}>
              <Box p={4}>
                <EmptyState message={Language.emptyMessage} />
              </Box>
            </TableCell>
          </TableRow>
        )}
      </TableBody>
    </Table>
  )
}
