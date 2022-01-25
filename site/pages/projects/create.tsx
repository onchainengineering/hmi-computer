import React from "react"
import { makeStyles } from "@material-ui/core/styles"
import { useRouter } from "next/router"
import useSWR from "swr"

import { provisioners } from "../../api"
import { useUser } from "../../contexts/UserContext"
import { FullScreenLoader } from "../../components/Loader/FullScreenLoader"
import { CreateProjectForm } from "../../forms/CreateProjectForm"

const CreateProjectPage: React.FC = () => {
  const router = useRouter()
  const styles = useStyles()
  const { me } = useUser(true)
  const { data: organizations, error } = useSWR("/api/v2/users/me/organizations")

  if (error) {
    // TODO: Merge with error component in other PR
    return <div>{"Error"}</div>
  }

  if (!me || !organizations) {
    return <FullScreenLoader />
  }

  const onCancel = async () => {
    await router.push("/projects")
  }

  return (
    <div className={styles.root}>
      <CreateProjectForm
        provisioners={provisioners}
        organizations={organizations}
        onSubmit={(request) => alert(JSON.stringify(request))}
        onCancel={onCancel}
      />
    </div>
  )
}

const useStyles = makeStyles((theme) => ({
  root: {
    display: "flex",
    flexDirection: "column",
    height: "100vh",
    backgroundColor: theme.palette.background.paper,
  },
}))

export default CreateProjectPage
