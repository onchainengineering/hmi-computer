import {
  BuildInfoResponse,
  Organization,
  Provisioner,
  Template,
  UserAgent,
  UserResponse,
  Workspace,
  Pager
} from "../api/types"
import { v4 as uuidv4 } from 'uuid';

export const MockSessionToken = { session_token: "my-session-token" }

export const MockAPIKey = { key: "my-api-key" }

export const MockBuildInfo: BuildInfoResponse = {
  external_url: "file:///mock-url",
  version: "v99.999.9999+c9cdf14",
}

export const MockUser: UserResponse = {
  id: "test-user",
  username: "TestUser",
  email: "test@coder.com",
  created_at: "",
}

export const MockUser2: UserResponse = {
  id: "test-user-2",
  username: "TestUser2",
  email: "test2@coder.com",
  created_at: "",
}

export const MockPager: Pager = {
  after: uuidv4(),
  before: uuidv4(),
  limit: 10,
  total: 25 
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

export const MockWorkspace: Workspace = {
  id: "test-workspace",
  name: "Test-Workspace",
  created_at: "",
  updated_at: "",
  template_id: MockTemplate.id,
  owner_id: MockUser.id,
}

export const MockUserAgent: UserAgent = {
  browser: "Chrome 99.0.4844",
  device: "Other",
  ip_address: "11.22.33.44",
  os: "Windows 10",
}
