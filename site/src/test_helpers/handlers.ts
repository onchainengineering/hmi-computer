import { rest } from "msw"
import * as M from "./entities"

export const handlers = [
  // build info
  rest.get("/api/v2/buildinfo", async (req, res, ctx) => {
    return res(ctx.status(200), ctx.json(M.MockBuildInfo))
  }),

  // organizations
  rest.get("/api/v2/organizations/:organizationId", async (req, res, ctx) => {
    return res(ctx.status(200), ctx.json(M.MockOrganization))
  }),
  rest.get("/api/v2/organizations/:organizationId/templates/:templateId", async (req, res, ctx) => {
    return res(ctx.status(200), ctx.json(M.MockTemplate))
  }),

  // templates
  rest.get("/api/v2/templates/:templateId", async (req, res, ctx) => {
    return res(ctx.status(200), ctx.json(M.MockTemplate))
  }),

  // users
  rest.post("/api/v2/users/me/workspaces", async (req, res, ctx) => {
    return res(ctx.status(200), ctx.json(M.MockWorkspace))
  }),
  rest.get("/api/v2/users/me/organizations/:organizationId", async (req, res, ctx) => {
    return res(ctx.status(200), ctx.json(M.MockOrganization))
  }),
  rest.post("/api/v2/users/login", async (req, res, ctx) => {
    return res(ctx.status(200), ctx.json(M.MockSessionToken))
  }),
  rest.post("/api/v2/users/logout", async (req, res, ctx) => {
    return res(ctx.status(200))
  }),
  rest.get("/api/v2/users/me", async (req, res, ctx) => {
    return res(ctx.status(200), ctx.json(M.MockUser))
  }),
  rest.get("/api/v2/users/me/keys", async (req, res, ctx) => {
    return res(ctx.status(200), ctx.json(M.MockAPIKey))
  }),

  // workspaces
  rest.get("/api/v2/workspaces/:workspaceId", async (req, res, ctx) => {
    return res(ctx.status(200), ctx.json(M.MockWorkspace))
  }),
  rest.get("/api/v2/workspaces/:workspaceId/autostart", async (req, res, ctx) => {
    return res(ctx.status(200))
  }),
  rest.get("/api/v2/workspaces/:workspaceId/autostop", async (req, res, ctx) => {
    return res(ctx.status(200))
  }),
]
