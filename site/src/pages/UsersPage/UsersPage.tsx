import { useActor } from "@xstate/react"
import React, { useContext, useEffect } from "react"
import { useNavigate } from "react-router"
import { ConfirmDialog } from "../../components/ConfirmDialog/ConfirmDialog"
import { ErrorSummary } from "../../components/ErrorSummary/ErrorSummary"
import { FullScreenLoader } from "../../components/Loader/FullScreenLoader"
import { XServiceContext } from "../../xServices/StateContext"
import { UsersPageView } from "./UsersPageView"

export const UsersPage: React.FC = () => {
  const xServices = useContext(XServiceContext)
  const [usersState, usersSend] = useActor(xServices.usersXService)
  const { users, getUsersError, userIdToSuspend } = usersState.context
  const navigate = useNavigate()
  const userToBeSuspended = users?.find((u) => u.id === userIdToSuspend)

  /**
   * Fetch users on component mount
   */
  useEffect(() => {
    usersSend("GET_USERS")
  }, [usersSend])

  if (usersState.matches("error")) {
    return <ErrorSummary error={getUsersError} />
  }

  if (!users) {
    return <FullScreenLoader />
  } else {
    return (
      <>
        <UsersPageView
          users={users}
          openUserCreationDialog={() => {
            navigate("/users/create")
          }}
          onSuspendUser={(user) => {
            usersSend({ type: "SUSPEND_USER", userId: user.id })
          }}
        />

        <ConfirmDialog
          type="delete"
          hideCancel={false}
          open={usersState.matches("confirmUserSuspension")}
          confirmLoading={usersState.matches("suspendingUser")}
          title="Suspend user"
          confirmText="Suspend"
          onConfirm={() => {
            usersSend("CONFIRM_USER_SUSPENSION")
          }}
          onClose={() => {
            usersSend("CANCEL_USER_SUSPENSION")
          }}
          description={
            <>
              Do you want to suspend the user <strong>{userToBeSuspended?.username}</strong>?
            </>
          }
        />
      </>
    )
  }
}
