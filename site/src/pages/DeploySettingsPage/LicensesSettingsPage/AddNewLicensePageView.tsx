import Button from "@material-ui/core/Button"
import TextField from "@material-ui/core/TextField"
import { makeStyles } from "@material-ui/core/styles"
import CloudUploadOutlined from "@material-ui/icons/CloudUploadOutlined"
import { Fieldset } from "components/DeploySettingsLayout/Fieldset"
import { Header } from "components/DeploySettingsLayout/Header"
import { Stack } from "components/Stack/Stack"
import { DropzoneDialog } from "material-ui-dropzone"
import { FC, PropsWithChildren, useState } from "react"
import { Link as RouterLink, useNavigate } from "react-router-dom"
import { useToggle } from "react-use"

const AddNewLicense: FC = () => {
  const styles = useStyles()
  const [isDialogOpen, toggleDialogOpen] = useToggle(false)
  const [files, setFiles] = useState<File[]>([])
  const navigate = useNavigate()

  function handleSave(files: File[]) {
    setFiles(files)
    toggleDialogOpen()
    navigate("/settings/deployment/licenses#success=true")
  }

  return (
    <>
      <Stack
        alignItems="baseline"
        direction="row"
        justifyContent="space-between"
      >
        <Header
          title="Add your license"
          description="Add a license to your account to unlock more features."
        />
        <Button
          component={RouterLink}
          to="/settings/deployment/licenses"
          variant="outlined"
        >
          Back to licenses
        </Button>
      </Stack>

      <Stack className={styles.main}>
        <Stack alignItems="center">
          <Button
            className={styles.ctaButton}
            startIcon={<CloudUploadOutlined />}
            size="large"
            onClick={() => toggleDialogOpen()}
          >
            Upload License File
          </Button>
        </Stack>
        <DropzoneDialog
          open={isDialogOpen}
          onSave={handleSave}
          // acceptedFiles={['image/jpeg', 'image/png', 'image/bmp']}
          showPreviews
          maxFileSize={1000000}
          onClose={() => toggleDialogOpen(false)}
        />

        <DividerWithText>or</DividerWithText>

        <Fieldset
          title="Paste your license key"
          onSubmit={(data: unknown) => {
            console.log(data)
          }}
        >
          <TextField
            placeholder="Paste your license key here"
            multiline
            rows={4}
            fullWidth
          />
        </Fieldset>
      </Stack>
    </>
  )
}

export default AddNewLicense

const useStyles = makeStyles((theme) => ({
  main: {
    paddingTop: theme.spacing(5),
  },
  ctaButton: {
    backgroundImage: `linear-gradient(90deg, ${theme.palette.secondary.main} 0%, ${theme.palette.secondary.dark} 100%)`,
    width: theme.spacing(30),
    marginBottom: theme.spacing(4),
  },
  formSectionRoot: {
    alignItems: "center",
  },
  description: {
    color: theme.palette.text.secondary,
    lineHeight: "160%",
  },
  title: {
    ...theme.typography.h5,
    fontWeight: 600,
    marging: theme.spacing(1),
  },
  container: {
    display: "flex",
    alignItems: "center",
  },
  border: {
    borderBottom: `2px solid ${theme.palette.divider}`,
    width: "100%",
  },
  content: {
    paddingTop: theme.spacing(0.5),
    paddingBottom: theme.spacing(0.5),
    paddingRight: theme.spacing(2),
    paddingLeft: theme.spacing(2),
    fontWeight: 500,
    fontSize: theme.typography.h5.fontSize,
    color: theme.palette.text.secondary,
  },
}))

const DividerWithText: FC<PropsWithChildren> = ({ children }) => {
  const classes = useStyles()
  return (
    <div className={classes.container}>
      <div className={classes.border} />
      <span className={classes.content}>{children}</span>
      <div className={classes.border} />
    </div>
  )
}
