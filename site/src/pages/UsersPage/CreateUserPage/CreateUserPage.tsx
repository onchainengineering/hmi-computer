import { useActor } from "@xstate/react"
import React, { useContext } from "react"
import { useNavigate } from "react-router"
import { isApiError, mapApiErrorToFieldErrors } from "../../../api/errors"
import { CreateUserRequest } from "../../../api/typesGenerated"
import { CreateUserForm } from "../../../components/CreateUserForm/CreateUserForm"
import { XServiceContext } from "../../../xServices/StateContext"

const Language = {
  unknownError: "Oops, an unknown error occurred.",
}

export const CreateUserPage = () => {
  const xServices = useContext(XServiceContext)
  const [usersState, usersSend] = useActor(xServices.usersXService)
  const { createUserError } = usersState.context
  const apiError = isApiError(createUserError)
  const formErrors = apiError ? mapApiErrorToFieldErrors(createUserError.response.data) : undefined
  const hasUnknownError = createUserError && !apiError
  const navigate = useNavigate()

  return (
    <CreateUserForm
      formErrors={formErrors}
      onSubmit={(user: CreateUserRequest) => usersSend({ type: "CREATE", user })}
      onCancel={() => navigate("/users")}
      isLoading={usersState.hasTag("loading")}
      error={hasUnknownError ? Language.unknownError : undefined}
    />
  )
}
