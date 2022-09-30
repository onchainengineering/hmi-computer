import CircularProgress from "@material-ui/core/CircularProgress"
import { makeStyles } from "@material-ui/core/styles"
import TextField from "@material-ui/core/TextField"
import Autocomplete from "@material-ui/lab/Autocomplete"
import { useMachine } from "@xstate/react"
import { Group, User } from "api/typesGenerated"
import { AvatarData } from "components/AvatarData/AvatarData"
import { useOrganizationId } from "hooks/useOrganizationId"
import debounce from "just-debounce-it"
import { ChangeEvent, useState } from "react"
import { searchUsersAndGroupsMachine } from "xServices/template/searchUsersAndGroupsXService"

export type UserOrGroupAutocompleteValue = User | Group | null

const isGroup = (value: UserOrGroupAutocompleteValue): value is Group => {
  return value !== null && "members" in value
}

export type UserOrGroupAutocompleteProps = {
  value: UserOrGroupAutocompleteValue
  onChange: (value: UserOrGroupAutocompleteValue) => void
}

export const UserOrGroupAutocomplete: React.FC<UserOrGroupAutocompleteProps> = ({
  value,
  onChange,
}) => {
  const styles = useStyles()
  const organizationId = useOrganizationId()
  const [isAutocompleteOpen, setIsAutocompleteOpen] = useState(false)
  const [searchState, sendSearch] = useMachine(searchUsersAndGroupsMachine, {
    context: {
      userResults: [],
      groupResults: [],
      organizationId,
    },
  })
  const { userResults, groupResults } = searchState.context
  const handleFilterChange = debounce((event: ChangeEvent<HTMLInputElement>) => {
    sendSearch("SEARCH", { query: event.target.value })
  }, 500)

  return (
    <Autocomplete
      value={value}
      id="user-or-group-autocomplete"
      open={isAutocompleteOpen}
      onOpen={() => {
        setIsAutocompleteOpen(true)
      }}
      onClose={() => {
        setIsAutocompleteOpen(false)
      }}
      onChange={(_, newValue) => {
        if (newValue === null) {
          sendSearch("CLEAR_RESULTS")
        }

        onChange(newValue)
      }}
      getOptionSelected={(option, value) => option.id === value.id}
      getOptionLabel={(option) => (isGroup(option) ? option.name : option.email)}
      renderOption={(option) => {
        const isOptionGroup = isGroup(option)

        return (
          <AvatarData
            title={isOptionGroup ? option.name : option.username}
            subtitle={isOptionGroup ? `${option.members.length} members` : option.email}
            highlightTitle
            avatar={
              !isOptionGroup && option.avatar_url ? (
                <img
                  className={styles.avatar}
                  alt={`${option.username}'s Avatar`}
                  src={option.avatar_url}
                />
              ) : null
            }
          />
        )
      }}
      options={[...groupResults, ...userResults]}
      loading={searchState.matches("searching")}
      className={styles.autocomplete}
      renderInput={(params) => (
        <TextField
          {...params}
          margin="none"
          variant="outlined"
          placeholder="Search for user or group"
          InputProps={{
            ...params.InputProps,
            onChange: handleFilterChange,
            endAdornment: (
              <>
                {searchState.matches("searching") ? <CircularProgress size={16} /> : null}
                {params.InputProps.endAdornment}
              </>
            ),
          }}
        />
      )}
    />
  )
}

export const useStyles = makeStyles((theme) => {
  return {
    autocomplete: {
      width: "300px",

      "& .MuiFormControl-root": {
        width: "100%",
      },

      "& .MuiInputBase-root": {
        width: "100%",
        // Match button small height
        height: 36,
      },

      "& input": {
        fontSize: 14,
        padding: `${theme.spacing(0, 0.5, 0, 0.5)} !important`,
      },
    },

    avatar: {
      width: theme.spacing(4.5),
      height: theme.spacing(4.5),
      borderRadius: "100%",
    },
  }
})
