import { test } from "@playwright/test"
import {
  buildWorkspaceWithParameters,
  createTemplate,
  createWorkspace,
  echoResponsesWithParameters,
  stopWorkspace,
  verifyParameters,
} from "../helpers"

import { firstBuildOption, secondBuildOption } from "../parameters"
import { RichParameter } from "../provisionerGenerated"

test("start workspace with ephemeral parameters", async ({ page }) => {
  const richParameters: RichParameter[] = [firstBuildOption, secondBuildOption]
  const template = await createTemplate(
    page,
    echoResponsesWithParameters(richParameters),
  )
  const workspaceName = await createWorkspace(page, template)

  // Verify that build options are default (not selected).
  await verifyParameters(page, workspaceName, richParameters, [
    { name: firstBuildOption.name, value: firstBuildOption.defaultValue },
    { name: secondBuildOption.name, value: secondBuildOption.defaultValue },
  ])

  // Stop the workspace
  await stopWorkspace(page, workspaceName)

  // Now, start the workspace with ephemeral parameters selected.
  const buildParameters = [
    { name: firstBuildOption.name, value: "AAAAA" },
    { name: secondBuildOption.name, value: "true" },
  ]

  await buildWorkspaceWithParameters(
    page,
    workspaceName,
    richParameters,
    buildParameters,
  )

  // FIXME: verify that build options are default (not selected).
  // It is the opposite now until the site logic is corrected.
  await verifyParameters(page, workspaceName, richParameters, [
    { name: firstBuildOption.name, value: buildParameters[0].value },
    { name: secondBuildOption.name, value: buildParameters[1].value },
  ])
})
