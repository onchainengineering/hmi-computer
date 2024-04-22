import type { QueryClient, UseQueryOptions } from "react-query";
import * as API from "api/api";
import type { AppearanceConfig } from "api/typesGenerated";
import { getMetadataAsJSON } from "utils/metadata";

const initialAppearanceData = getMetadataAsJSON<AppearanceConfig>("appearance");
const appearanceConfigKey = ["appearance"] as const;

export const appearance = (): UseQueryOptions<AppearanceConfig> => {
  return {
    // We either have our initial data or should immediately
    // fetch and never again!
    cacheTime: Infinity,
    staleTime: Infinity,
    refetchOnMount: false,
    refetchOnReconnect: false,
    refetchOnWindowFocus: false,
    queryKey: ["appearance"],
    initialData: initialAppearanceData,
    queryFn: () => API.getAppearance(),
  };
};

export const updateAppearance = (queryClient: QueryClient) => {
  return {
    mutationFn: API.updateAppearance,
    onSuccess: (newConfig: AppearanceConfig) => {
      queryClient.setQueryData(appearanceConfigKey, newConfig);
    },
  };
};
