import { useActor } from "@xstate/react"
import React, { useContext, useEffect } from "react"
import { useNavigate } from "react-router"
import { ConfirmDialog } from "../../components/ConfirmDialog/ConfirmDialog"
import { FullScreenLoader } from "../../components/Loader/FullScreenLoader"
import { ResetPasswordDialog } from "../../components/ResetPasswordDialog/ResetPasswordDialog"
import { XServiceContext } from "../../xServices/StateContext"
import { UsersPageView } from "./UsersPageView"

export const Language = {
  suspendDialogTitle: "Suspend user",
  suspendDialogAction: "Suspend",
  suspendDialogMessagePrefix: "Do you want to suspend the user",
}

const useRoles = () => {
  const xServices = useContext(XServiceContext)
  const [authState] = useActor(xServices.authXService)
  const [rolesState, rolesSend] = useActor(xServices.siteRolesXService)
  const { roles } = rolesState.context
  const { me } = authState.context

  useEffect(() => {
    if (!me) {
      throw new Error("User is not logged in")
    }

    const organizationId = me.organization_ids[0]

    rolesSend({
      type: "GET_ROLES",
      organizationId,
    })
  }, [me, rolesSend])

  return roles
}

export const UsersPage: React.FC = () => {
  const xServices = useContext(XServiceContext)
  const [usersState, usersSend] = useActor(xServices.usersXService)
  const { users, getUsersError, userIdToSuspend, userIdToResetPassword, newUserPassword } = usersState.context
  const navigate = useNavigate()
  const userToBeSuspended = users?.find((u) => u.id === userIdToSuspend)
  const userToResetPassword = users?.find((u) => u.id === userIdToResetPassword)
  const roles = useRoles()

  /**
   * Fetch users on component mount
   */
  useEffect(() => {
    usersSend("GET_USERS")
  }, [usersSend])

  if (!users || !roles) {
    return <FullScreenLoader />
  } else {
    return (
      <>
        <UsersPageView
          roles={roles}
          users={users}
          openUserCreationDialog={() => {
            navigate("/users/create")
          }}
          onSuspendUser={(user) => {
            usersSend({ type: "SUSPEND_USER", userId: user.id })
          }}
          onResetUserPassword={(user) => {
            usersSend({ type: "RESET_USER_PASSWORD", userId: user.id })
          }}
          onUpdateUserRoles={(user, roles) => {
            usersSend({
              type: "UPDATE_USER_ROLES",
              userId: user.id,
              roles,
            })
          }}
          error={getUsersError}
          isUpdatingUserRoles={usersState.matches("updatingUserRoles")}
        />

        <ConfirmDialog
          type="delete"
          hideCancel={false}
          open={usersState.matches("confirmUserSuspension")}
          confirmLoading={usersState.matches("suspendingUser")}
          title={Language.suspendDialogTitle}
          confirmText={Language.suspendDialogAction}
          onConfirm={() => {
            usersSend("CONFIRM_USER_SUSPENSION")
          }}
          onClose={() => {
            usersSend("CANCEL_USER_SUSPENSION")
          }}
          description={
            <>
              {Language.suspendDialogMessagePrefix} <strong>{userToBeSuspended?.username}</strong>?
            </>
          }
        />

        <ResetPasswordDialog
          loading={usersState.matches("resettingUserPassword")}
          user={userToResetPassword}
          newPassword={newUserPassword}
          open={usersState.matches("confirmUserPasswordReset")}
          onClose={() => {
            usersSend("CANCEL_USER_PASSWORD_RESET")
          }}
          onConfirm={() => {
            usersSend("CONFIRM_USER_PASSWORD_RESET")
          }}
        />
      </>
    )
  }
}
