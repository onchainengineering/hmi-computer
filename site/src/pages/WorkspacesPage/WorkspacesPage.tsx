import Button from "@material-ui/core/Button"
import Link from "@material-ui/core/Link"
import Menu from "@material-ui/core/Menu"
import MenuItem from "@material-ui/core/MenuItem"
import { makeStyles } from "@material-ui/core/styles"
import TextField from "@material-ui/core/TextField"
import AddCircleOutline from "@material-ui/icons/AddCircleOutline"
import { useMachine } from "@xstate/react"
import { FormikErrors, useFormik } from "formik"
import { FC, useState } from "react"
import { Link as RouterLink } from "react-router-dom"
import { CloseDropdown, OpenDropdown } from "../../components/DropdownArrows/DropdownArrows"
import { Margins } from "../../components/Margins/Margins"
import { Stack } from "../../components/Stack/Stack"
import { getFormHelpers, onChangeTrimmed } from "../../util/formUtils"
import { workspacesMachine } from "../../xServices/workspaces/workspacesXService"
import { WorkspacesPageView } from "./WorkspacesPageView"

interface FilterFormValues {
  query: string
}

const Language = {
  filterName: "Filters",
  createWorkspaceButton: "Create workspace",
  yourWorkspacesButton: "Your workspaces",
  allWorkspacesButton: "All workspaces",
}

export type FilterFormErrors = FormikErrors<FilterFormValues>

const WorkspacesPage: FC = () => {
  const styles = useStyles()
  const [workspacesState, send] = useMachine(workspacesMachine)

  const form = useFormik<FilterFormValues>({
    initialValues: {
      query: workspacesState.context.filter || "",
    },
    onSubmit: (values) => {
      send({
        type: "SET_FILTER",
        query: values.query,
      })
    },
  })

  const getFieldHelpers = getFormHelpers<FilterFormValues>(form)

  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null)

  const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    setAnchorEl(event.currentTarget)
  }

  const handleClose = () => {
    setAnchorEl(null)
  }

  const setYourWorkspaces = () => {
    void form.setFieldValue("query", "owner:me")
    void form.submitForm()
    handleClose()
  }

  const setAllWorkspaces = () => {
    void form.setFieldValue("query", "")
    void form.submitForm()
    handleClose()
  }

  return (
    <>
      <Margins>
        <div className={styles.actions}>
          <Stack direction="row">
            <Button aria-controls="filter-menu" aria-haspopup="true" onClick={handleClick}>
              {Language.filterName} {anchorEl ? <CloseDropdown /> : <OpenDropdown />}
            </Button>

            <Menu id="filter-menu" anchorEl={anchorEl} keepMounted open={Boolean(anchorEl)} onClose={handleClose}>
              <MenuItem onClick={setYourWorkspaces}>{Language.yourWorkspacesButton}</MenuItem>
              <MenuItem onClick={setAllWorkspaces}>{Language.allWorkspacesButton}</MenuItem>
            </Menu>

            <form onSubmit={form.handleSubmit}>
              <TextField {...getFieldHelpers("query")} onChange={onChangeTrimmed(form)} fullWidth variant="outlined" />
            </form>
          </Stack>

          <Link underline="none" component={RouterLink} to="/workspaces/new">
            <Button startIcon={<AddCircleOutline />}>{Language.createWorkspaceButton}</Button>
          </Link>
        </div>

        <WorkspacesPageView
          loading={workspacesState.hasTag("loading")}
          workspaces={workspacesState.context.workspaces}
        />
      </Margins>
    </>
  )
}

const useStyles = makeStyles((theme) => ({
  actions: {
    marginTop: theme.spacing(3),
    marginBottom: theme.spacing(3),
    display: "flex",
    justifyContent: "space-between",
    height: theme.spacing(6),
  },
}))

export default WorkspacesPage
