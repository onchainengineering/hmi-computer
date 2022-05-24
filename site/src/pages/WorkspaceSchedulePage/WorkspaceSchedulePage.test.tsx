import * as TypesGen from "../../api/typesGenerated"
import { WorkspaceScheduleFormValues } from "../../components/WorkspaceStats/WorkspaceScheduleForm"
import { formValuesToAutoStartRequest, formValuesToTTLRequest } from "./WorkspaceSchedulePage"

const validValues: WorkspaceScheduleFormValues = {
  sunday: false,
  monday: true,
  tuesday: true,
  wednesday: true,
  thursday: true,
  friday: true,
  saturday: false,
  startTime: "09:30",
  ttl: 120,
}

describe("WorkspaceSchedulePage", () => {
  describe("formValuesToAutoStartRequest", () => {
    it.each<[WorkspaceScheduleFormValues, TypesGen.UpdateWorkspaceAutostartRequest]>([
      [
        // Empty case
        {
          sunday: false,
          monday: false,
          tuesday: false,
          wednesday: false,
          thursday: false,
          friday: false,
          saturday: false,
          startTime: "",
          ttl: 0,
        },
        {
          schedule: "",
        },
      ],
      [
        // Single day
        {
          sunday: true,
          monday: false,
          tuesday: false,
          wednesday: false,
          thursday: false,
          friday: false,
          saturday: false,
          startTime: "16:20",
          ttl: 120,
        },
        {
          schedule: "20 16 * * 0",
        },
      ],
      [
        // Standard 1-5 case
        {
          sunday: false,
          monday: true,
          tuesday: true,
          wednesday: true,
          thursday: true,
          friday: true,
          saturday: false,
          startTime: "09:30",
          ttl: 120,
        },
        {
          schedule: "30 09 * * 1-5",
        },
      ],
      [
        // Everyday
        {
          sunday: true,
          monday: true,
          tuesday: true,
          wednesday: true,
          thursday: true,
          friday: true,
          saturday: true,
          startTime: "09:00",
          ttl: 60 * 8,
        },
        {
          schedule: "00 09 * * 1-7",
        },
      ],
      [
        // Mon, Wed, Fri Evenings
        {
          sunday: false,
          monday: true,
          tuesday: false,
          wednesday: true,
          thursday: false,
          friday: true,
          saturday: false,
          startTime: "16:20",
          ttl: 60 * 3,
        },
        {
          schedule: "20 16 * * 1,3,5",
        },
      ],
    ])(`formValuesToAutoStartRequest(%p) return %p`, (values, request) => {
      expect(formValuesToAutoStartRequest(values)).toEqual(request)
    })
  })

  describe("formValuesToTTLRequest", () => {
    it.each<[WorkspaceScheduleFormValues, TypesGen.UpdateWorkspaceTTLRequest]>([
      [
        // 0 case
        {
          ...validValues,
          ttl: 0,
        },
        {
          ttl: undefined,
        },
      ],
      [
        // 2 Hours = 7.2e+12 case
        {
          ...validValues,
          ttl: 120,
        },
        {
          ttl: 7_200_000_000_000,
        },
      ],
      [
        // 8 hours = 2.88e+13 case
        {
          ...validValues,
          ttl: 60 * 8,
        },
        {
          ttl: 28_800_000_000_000,
        },
      ],
    ])(`formValuesToTTLRequest(%p) returns %p`, (values, request) => {
      expect(formValuesToTTLRequest(values)).toEqual(request)
    })
  })
})
