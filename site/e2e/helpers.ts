import { Page } from "@playwright/test"
import path from "path"

export const buttons = {
  starterTemplates: "Starter Templates",
  dockerTemplate: "Develop in Docker",
  useTemplate: "Create Workspace",
  createTemplate: "Create Template",
  createWorkspace: "Create Workspace",
  submitCreateWorkspace: "Create Workspace",
  stopWorkspace: "Stop",
  startWorkspace: "Start",
}

export const clickButton = async (page: Page, name: string): Promise<void> => {
  await page.getByRole("button", { name, exact: true }).click()
}

export const fillInput = async (
  page: Page,
  label: string,
  value: string,
): Promise<void> => {
  await page.fill(`text=${label}`, value)
}

const statesDir = path.join(__dirname, "./states")

export const getStatePath = (name: string): string => {
  return path.join(statesDir, `${name}.json`)
}
