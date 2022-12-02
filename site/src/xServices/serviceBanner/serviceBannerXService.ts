import { assign, createMachine } from "xstate"
import * as API from "../../api/api"
import { ServiceBanner } from "../../api/typesGenerated"

export const Language = {
  getServiceBannerError: "Error getting service banner.",
}

export type ServiceBannerContext = {
  serviceBanner: ServiceBanner
  getServiceBannerError?: Error | unknown
}

export type ServiceBannerEvent =
  | {
      type: "GET_BANNER"
    }
  | { type: "SET_PREVIEW"; serviceBanner: ServiceBanner }

const emptyBanner = {
  enabled: false,
}

export const serviceBannerMachine = createMachine(
  {
    id: "serviceBannerMachine",
    predictableActionArguments: true,
    tsTypes: {} as import("./serviceBannerXService.typegen").Typegen0,
    schema: {
      context: {} as ServiceBannerContext,
      events: {} as ServiceBannerEvent,
      services: {
        getServiceBanner: {
          data: {} as ServiceBanner,
        },
      },
    },
    context: {
      serviceBanner: emptyBanner,
    },
    initial: "idle",
    states: {
      idle: {
        on: {
          GET_BANNER: "gettingBanner",
          SET_PREVIEW: "settingPreview",
        },
      },
      gettingBanner: {
        entry: "clearGetBannerError",
        invoke: {
          id: "getBanner",
          src: "getBanner",
          onDone: {
            target: "idle",
            actions: ["assignBanner"],
          },
          onError: {
            target: "idle",
            actions: ["assignGetBannerError"],
          },
        },
      },
      settingPreview: {
        entry: ["clearGetBannerError", "assignPreviewBanner"],
        always: {
          target: "idle",
        },
      },
    },
  },
  {
    actions: {
      assignPreviewBanner: assign({
        serviceBanner: (_, event) => event.serviceBanner,
      }),
      assignBanner: assign({
        serviceBanner: (_, event) => event.data as ServiceBanner,
      }),
      assignGetBannerError: assign({
        getServiceBannerError: (_, event) => event.data,
      }),
      clearGetBannerError: assign({
        getServiceBannerError: (_) => undefined,
      }),
    },
    services: {
      getBanner: API.getServiceBanner,
    },
  },
)
