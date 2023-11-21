import { getAuditLogs } from "api/api";
import { type AuditLogResponse } from "api/typesGenerated";
import { useFilterParamsKey } from "components/Filter/filter";
import { type UsePaginatedQueryOptions } from "hooks/usePaginatedQuery";

export function paginatedAudits(searchParams: URLSearchParams) {
  return {
    searchParams,
    queryPayload: () => searchParams.get(useFilterParamsKey) ?? "",
    queryKey: ({ payload, pageNumber }) => {
      return ["auditLogs", payload, pageNumber] as const;
    },
    queryFn: ({ payload, limit, offset }) => {
      return getAuditLogs({
        offset,
        limit,
        q: payload,
      });
    },

    cacheTime: 5 * 1000 * 60,
  } as const satisfies UsePaginatedQueryOptions<AuditLogResponse, string>;
}
