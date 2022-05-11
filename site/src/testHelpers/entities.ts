import {
  BuildInfoResponse,
  Organization,
  Provisioner,
  Template,
  UserAgent,
  UserResponse,
  Workspace,
  WorkspaceAgent,
  WorkspaceAutostartRequest,
  WorkspaceBuildTransition,
  WorkspaceResource,
} from "../api/types"
import { AuthMethods, ProvisionerJobStatus, Role } from "../api/typesGenerated"

export const MockSessionToken = { session_token: "my-session-token" }

export const MockAPIKey = { key: "my-api-key" }

export const MockBuildInfo: BuildInfoResponse = {
  external_url: "file:///mock-url",
  version: "v99.999.9999+c9cdf14",
}

export const MockAdminRole: Role = {
  name: "admin",
  display_name: "Admin",
}

export const MockMemberRole: Role = {
  name: "member",
  display_name: "Member",
}

export const MockAuditorRole: Role = {
  name: "auditor",
  display_name: "Auditor",
}

export const MockSiteRoles = [MockAdminRole, MockAuditorRole, MockMemberRole]

export const MockUser: UserResponse = {
  id: "test-user",
  username: "TestUser",
  email: "test@coder.com",
  created_at: "",
  status: "active",
  organization_ids: ["fc0774ce-cc9e-48d4-80ae-88f7a4d4a8b0"],
  roles: [MockAdminRole, MockMemberRole],
}

export const MockUser2: UserResponse = {
  id: "test-user-2",
  username: "TestUser2",
  email: "test2@coder.com",
  created_at: "",
  status: "active",
  organization_ids: ["fc0774ce-cc9e-48d4-80ae-88f7a4d4a8b0"],
  roles: [MockMemberRole],
}

export const MockOrganization: Organization = {
  id: "test-org",
  name: "Test Organization",
  created_at: "",
  updated_at: "",
}

export const MockProvisioner: Provisioner = {
  id: "test-provisioner",
  name: "Test Provisioner",
}

export const MockTemplate: Template = {
  id: "test-template",
  created_at: "",
  updated_at: "",
  organization_id: MockOrganization.id,
  name: "Test Template",
  provisioner: MockProvisioner.id,
  active_version_id: "",
}

export const MockWorkspaceAutostartDisabled: WorkspaceAutostartRequest = {
  schedule: "",
}

export const MockWorkspaceAutostartEnabled: WorkspaceAutostartRequest = {
  // Runs at 9:30am Monday through Friday using Canada/Eastern
  // (America/Toronto) time
  schedule: "CRON_TZ=Canada/Eastern 30 9 * * 1-5",
}

export const MockWorkspaceAutostopDisabled: WorkspaceAutostartRequest = {
  schedule: "",
}

export const MockWorkspaceAutostopEnabled: WorkspaceAutostartRequest = {
  // Runs at 9:30pm Monday through Friday using America/Toronto
  schedule: "CRON_TZ=America/Toronto 30 21 * * 1-5",
}

export const MockProvisionerJob = {
  id: "test-provisioner-job",
  created_at: "",
  started_at: "",
  completed_at: "",
  error: "",
  status: "succeeded" as ProvisionerJobStatus,
  worker_id: "test-worker-id",
}

export const MockFailedProvisionerJob = {
  id: "test-provisioner-job",
  created_at: "",
  started_at: "",
  completed_at: "",
  error: "",
  status: "failed" as ProvisionerJobStatus,
  worker_id: "test-worker-id",
}

export const MockWorkspaceBuild = {
  id: "test-workspace-build",
  transition: "start" as WorkspaceBuildTransition,
  job: MockProvisionerJob,
}

export const MockWorkspaceBuildStop = {
  id: "test-workspace-build",
  transition: "stop" as WorkspaceBuildTransition,
  job: MockProvisionerJob,
}

export const MockFailedWorkspaceBuild = {
  id: "test-workspace-build",
  transition: "start" as WorkspaceBuildTransition,
  job: MockFailedProvisionerJob,
}

// These are special cases of MockWorkspaceBuild for more precise testing
export const MockWorkspaceStart = {
  id: "test-workspace-build-start",
  transition: "start",
}

export const MockWorkspaceStop = {
  id: "test-workspace-build-stop",
  transition: "stop",
}

export const MockWorkspaceDelete = {
  id: "test-workspace-build-delete",
  transition: "delete",
}

export const MockWorkspace: Workspace = {
  id: "test-workspace",
  name: "Test-Workspace",
  created_at: "",
  updated_at: "",
  template_id: MockTemplate.id,
  outdated: false,
  owner_id: MockUser.id,
  autostart_schedule: MockWorkspaceAutostartEnabled.schedule,
  autostop_schedule: MockWorkspaceAutostopEnabled.schedule,
  latest_build: MockWorkspaceBuild,
}

export const MockStoppedWorkspace: Workspace = {
  id: "test-workspace",
  name: "Test-Workspace",
  created_at: "",
  updated_at: "",
  template_id: MockTemplate.id,
  outdated: false,
  owner_id: MockUser.id,
  autostart_schedule: MockWorkspaceAutostartEnabled.schedule,
  autostop_schedule: MockWorkspaceAutostopEnabled.schedule,
  latest_build: MockWorkspaceBuildStop,
}

export const MockFailedWorkspace: Workspace = {
  id: "test-workspace",
  name: "Test-Workspace",
  created_at: "",
  updated_at: "",
  template_id: MockTemplate.id,
  outdated: false,
  owner_id: MockUser.id,
  autostart_schedule: MockWorkspaceAutostartEnabled.schedule,
  autostop_schedule: MockWorkspaceAutostopEnabled.schedule,
  latest_build: MockFailedWorkspaceBuild,
}

export const MockWorkspaceAgent: WorkspaceAgent = {
  id: "test-workspace-agent",
  name: "a-workspace-agent",
  operating_system: "linux",
}

export const MockWorkspaceResource: WorkspaceResource = {
  id: "test-workspace-resource",
  agents: [MockWorkspaceAgent],
}

export const MockUserAgent: UserAgent = {
  browser: "Chrome 99.0.4844",
  device: "Other",
  ip_address: "11.22.33.44",
  os: "Windows 10",
}

export const MockAuthMethods: AuthMethods = {
  password: true,
  github: false,
}
