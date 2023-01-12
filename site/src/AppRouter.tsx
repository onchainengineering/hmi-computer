import { useSelector } from "@xstate/react"
import { FeatureNames } from "api/types"
import { FullScreenLoader } from "components/Loader/FullScreenLoader"
import { RequirePermission } from "components/RequirePermission/RequirePermission"
import { TemplateLayout } from "components/TemplateLayout/TemplateLayout"
import { UsersLayout } from "components/UsersLayout/UsersLayout"
import IndexPage from "pages"
import AuditPage from "pages/AuditPage/AuditPage"
import GroupsPage from "pages/GroupsPage/GroupsPage"
import LoginPage from "pages/LoginPage/LoginPage"
import { SetupPage } from "pages/SetupPage/SetupPage"
import { TemplateSettingsPage } from "pages/TemplateSettingsPage/TemplateSettingsPage"
import TemplatesPage from "pages/TemplatesPage/TemplatesPage"
import UsersPage from "pages/UsersPage/UsersPage"
import WorkspacesPage from "pages/WorkspacesPage/WorkspacesPage"
import { FC, lazy, Suspense, useContext } from "react"
import { Route, Routes } from "react-router-dom"
import { selectPermissions } from "xServices/auth/authSelectors"
import { selectFeatureVisibility } from "xServices/entitlements/entitlementsSelectors"
import { XServiceContext } from "xServices/StateContext"
import { DashboardLayout } from "./components/DashboardLayout/DashboardLayout"
import { RequireAuth } from "./components/RequireAuth/RequireAuth"
import { SettingsLayout } from "./components/SettingsLayout/SettingsLayout"
import { DeploySettingsLayout } from "components/DeploySettingsLayout/DeploySettingsLayout"

// Lazy load pages
// - Pages that are secondary, not in the main navigation or not usually accessed
// - Pages that use heavy dependencies like charts or time libraries
const NotFoundPage = lazy(() => import("./pages/404Page/404Page"))
const CliAuthenticationPage = lazy(
  () => import("./pages/CliAuthPage/CliAuthPage"),
)
const AccountPage = lazy(
  () => import("./pages/UserSettingsPage/AccountPage/AccountPage"),
)
const SecurityPage = lazy(
  () => import("./pages/UserSettingsPage/SecurityPage/SecurityPage"),
)
const SSHKeysPage = lazy(
  () => import("./pages/UserSettingsPage/SSHKeysPage/SSHKeysPage"),
)
const CreateUserPage = lazy(
  () => import("./pages/UsersPage/CreateUserPage/CreateUserPage"),
)
const WorkspaceBuildPage = lazy(
  () => import("./pages/WorkspaceBuildPage/WorkspaceBuildPage"),
)
const WorkspacePage = lazy(() => import("./pages/WorkspacePage/WorkspacePage"))
const WorkspaceChangeVersionPage = lazy(
  () => import("./pages/WorkspaceChangeVersionPage/WorkspaceChangeVersionPage"),
)
const WorkspaceSchedulePage = lazy(
  () => import("./pages/WorkspaceSchedulePage/WorkspaceSchedulePage"),
)
const TerminalPage = lazy(() => import("./pages/TerminalPage/TerminalPage"))
const TemplatePermissionsPage = lazy(
  () =>
    import(
      "./pages/TemplatePage/TemplatePermissionsPage/TemplatePermissionsPage"
    ),
)
const TemplateSummaryPage = lazy(
  () => import("./pages/TemplatePage/TemplateSummaryPage/TemplateSummaryPage"),
)
const CreateWorkspacePage = lazy(
  () => import("./pages/CreateWorkspacePage/CreateWorkspacePage"),
)
const CreateGroupPage = lazy(() => import("./pages/GroupsPage/CreateGroupPage"))
const GroupPage = lazy(() => import("./pages/GroupsPage/GroupPage"))
const SettingsGroupPage = lazy(
  () => import("./pages/GroupsPage/SettingsGroupPage"),
)
const GeneralSettingsPage = lazy(
  () =>
    import(
      "./pages/DeploySettingsPage/GeneralSettingsPage/GeneralSettingsPage"
    ),
)
const SecuritySettingsPage = lazy(
  () =>
    import(
      "./pages/DeploySettingsPage/SecuritySettingsPage/SecuritySettingsPage"
    ),
)
const AppearanceSettingsPage = lazy(
  () =>
    import(
      "./pages/DeploySettingsPage/AppearanceSettingsPage/AppearanceSettingsPage"
    ),
)
const UserAuthSettingsPage = lazy(
  () =>
    import(
      "./pages/DeploySettingsPage/UserAuthSettingsPage/UserAuthSettingsPage"
    ),
)
const GitAuthSettingsPage = lazy(
  () =>
    import(
      "./pages/DeploySettingsPage/GitAuthSettingsPage/GitAuthSettingsPage"
    ),
)
const NetworkSettingsPage = lazy(
  () =>
    import(
      "./pages/DeploySettingsPage/NetworkSettingsPage/NetworkSettingsPage"
    ),
)
const GitAuthPage = lazy(() => import("./pages/GitAuthPage/GitAuthPage"))
const TemplateVersionPage = lazy(
  () => import("./pages/TemplateVersionPage/TemplateVersionPage"),
)
const StarterTemplatesPage = lazy(
  () => import("./pages/StarterTemplatesPage/StarterTemplatesPage"),
)
const StarterTemplatePage = lazy(
  () => import("pages/StarterTemplatePage/StarterTemplatePage"),
)
const CreateTemplatePage = lazy(
  () => import("./pages/CreateTemplatePage/CreateTemplatePage"),
)

export const AppRouter: FC = () => {
  const xServices = useContext(XServiceContext)
  const permissions = useSelector(xServices.authXService, selectPermissions)
  const featureVisibility = useSelector(
    xServices.entitlementsXService,
    selectFeatureVisibility,
  )

  return (
    <Suspense fallback={<FullScreenLoader />}>
      <Routes>
        <Route path="login" element={<LoginPage />} />
        <Route path="setup" element={<SetupPage />} />

        {/* Dashboard routes */}
        <Route element={<RequireAuth />}>
          <Route element={<DashboardLayout />}>
            <Route index element={<IndexPage />} />

            <Route path="cli-auth" element={<CliAuthenticationPage />} />
            <Route path="gitauth" element={<GitAuthPage />} />

            <Route path="workspaces" element={<WorkspacesPage />} />

            <Route path="starter-templates">
              <Route index element={<StarterTemplatesPage />} />
              <Route path=":exampleId" element={<StarterTemplatePage />} />
            </Route>

            <Route path="templates">
              <Route index element={<TemplatesPage />} />
              <Route path="new" element={<CreateTemplatePage />} />
              <Route path=":template">
                <Route
                  index
                  element={
                    <TemplateLayout>
                      <TemplateSummaryPage />
                    </TemplateLayout>
                  }
                />
                <Route
                  path="permissions"
                  element={
                    <TemplateLayout>
                      <TemplatePermissionsPage />
                    </TemplateLayout>
                  }
                />
                <Route path="workspace" element={<CreateWorkspacePage />} />
                <Route path="settings" element={<TemplateSettingsPage />} />
                <Route path="versions">
                  <Route path=":version" element={<TemplateVersionPage />} />
                </Route>
              </Route>
            </Route>

            <Route path="users">
              <Route
                index
                element={
                  <UsersLayout>
                    <UsersPage />
                  </UsersLayout>
                }
              />
              <Route path="create" element={<CreateUserPage />} />
            </Route>

            <Route path="/groups">
              <Route
                index
                element={
                  <UsersLayout>
                    <GroupsPage />
                  </UsersLayout>
                }
              />
              <Route path="create" element={<CreateGroupPage />} />
              <Route path=":groupId" element={<GroupPage />} />
              <Route path=":groupId/settings" element={<SettingsGroupPage />} />
            </Route>

            <Route path="/audit">
              <Route
                index
                element={
                  <RequirePermission
                    isFeatureVisible={
                      featureVisibility[FeatureNames.AuditLog] &&
                      Boolean(permissions?.viewAuditLog)
                    }
                  >
                    <AuditPage />
                  </RequirePermission>
                }
              />
            </Route>

            <Route path="/settings/deployment">
              <Route
                path="general"
                element={
                  <RequirePermission
                    isFeatureVisible={Boolean(
                      permissions?.viewDeploymentConfig,
                    )}
                  >
                    <DeploySettingsLayout>
                      <GeneralSettingsPage />
                    </DeploySettingsLayout>
                  </RequirePermission>
                }
              />
              <Route
                path="security"
                element={
                  <RequirePermission
                    isFeatureVisible={Boolean(
                      permissions?.viewDeploymentConfig,
                    )}
                  >
                    <DeploySettingsLayout>
                      <SecuritySettingsPage />
                    </DeploySettingsLayout>
                  </RequirePermission>
                }
              />
              <Route
                path="appearance"
                element={
                  <RequirePermission
                    isFeatureVisible={Boolean(
                      permissions?.viewDeploymentConfig,
                    )}
                  >
                    <DeploySettingsLayout>
                      <AppearanceSettingsPage />
                    </DeploySettingsLayout>
                  </RequirePermission>
                }
              />
              <Route
                path="network"
                element={
                  <RequirePermission
                    isFeatureVisible={Boolean(
                      permissions?.viewDeploymentConfig,
                    )}
                  >
                    <DeploySettingsLayout>
                      <NetworkSettingsPage />
                    </DeploySettingsLayout>
                  </RequirePermission>
                }
              />
              <Route
                path="userauth"
                element={
                  <RequirePermission
                    isFeatureVisible={Boolean(
                      permissions?.viewDeploymentConfig,
                    )}
                  >
                    <DeploySettingsLayout>
                      <UserAuthSettingsPage />
                    </DeploySettingsLayout>
                  </RequirePermission>
                }
              />
              <Route
                path="gitauth"
                element={
                  <RequirePermission
                    isFeatureVisible={Boolean(
                      permissions?.viewDeploymentConfig,
                    )}
                  >
                    <DeploySettingsLayout>
                      <GitAuthSettingsPage />
                    </DeploySettingsLayout>
                  </RequirePermission>
                }
              />
            </Route>

            <Route path="settings">
              <Route
                path="account"
                element={
                  <SettingsLayout>
                    <AccountPage />
                  </SettingsLayout>
                }
              />
              <Route
                path="security"
                element={
                  <SettingsLayout>
                    <SecurityPage />
                  </SettingsLayout>
                }
              />
              <Route
                path="ssh-keys"
                element={
                  <SettingsLayout>
                    <SSHKeysPage />
                  </SettingsLayout>
                }
              />
            </Route>

            <Route path="/@:username">
              <Route path=":workspace">
                <Route index element={<WorkspacePage />} />
                <Route path="schedule" element={<WorkspaceSchedulePage />} />
                <Route path="terminal" element={<TerminalPage />} />
                <Route
                  path="builds/:buildNumber"
                  element={<WorkspaceBuildPage />}
                />
                <Route
                  path="change-version"
                  element={<WorkspaceChangeVersionPage />}
                />
              </Route>
            </Route>
          </Route>
        </Route>

        {/* Using path="*"" means "match anything", so this route
        acts like a catch-all for URLs that we don't have explicit
        routes for. */}
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </Suspense>
  )
}
