"use client"

import { useInfiniteQuery, useQueryClient, keepPreviousData } from "@tanstack/react-query"

import {
  useListTenders,
  useGetTender,
  useGetTenderFilters,
  useAddTenderFavourite,
  useRemoveTenderFavourite,
  useGenerateTenderBrief as useGenerateTenderBrief_,
  useGenerateTenderDeliveryTimeline as useGenerateTenderDeliveryTimeline_,
  useGenerateTenderMatch as useGenerateTenderMatch_,
  listTenders,
} from "@/lib/api/generated/tenders/tenders"
import { ListTendersParams } from "@/lib/api/generated/model"

// Shared by any mutation that should make list/detail tender views refetch.
function invalidateTenderQueries(queryClient: ReturnType<typeof useQueryClient>) {
  return queryClient.invalidateQueries({
    predicate: (query) => {
      const key = query.queryKey[0]
      return typeof key === "string" && (key === "tenders" || key.startsWith("/api/tenders"))
    },
  })
}

const INFINITE_PAGE_SIZE = 12

// Read-only, paginated list of GeBIZ tenders. Thin wrapper over the generated
// query hook so the page imports from features/ like the other modules.
export function useTenders(params: ListTendersParams, enabled = true) {
  const { data, isLoading, error } = useListTenders(params, { query: { enabled } })
  return {
    items: data?.items ?? [],
    total: data?.total ?? 0,
    isLoading,
    error,
  }
}

// List-view variant: loads pages on demand instead of via page number, for
// scroll-triggered lazy loading. Calls the generated fetch function directly
// since orval only emits a plain useQuery hook, not a useInfiniteQuery one.
export function useTendersInfinite(
  params: Omit<ListTendersParams, "page" | "pageSize">,
  enabled = true,
) {
  const query = useInfiniteQuery({
    queryKey: ["tenders", "infinite", params],
    queryFn: ({ pageParam, signal }) =>
      listTenders({ ...params, page: pageParam, pageSize: INFINITE_PAGE_SIZE }, signal),
    initialPageParam: 1,
    getNextPageParam: (lastPage, allPages) =>
      allPages.length * INFINITE_PAGE_SIZE < lastPage.total ? allPages.length + 1 : undefined,
    enabled,
  })
  return {
    items: query.data?.pages.flatMap((p) => p.items) ?? [],
    isLoading: query.isLoading,
    isLoadingMore: query.isFetchingNextPage,
    hasMore: !!query.hasNextPage,
    loadMore: query.fetchNextPage,
    error: query.error,
  }
}

const TENDER_POLL_INTERVAL_MS = 2000
const TENDER_POLL_MAX_ATTEMPTS = 8

// GeBIZ detail is scraped in the background on first view, so poll briefly until it
// lands. AI brief/timeline are generated via separate explicit endpoints (see
// useGenerateTenderBrief/useGenerateTenderDeliveryTimeline below), not polled here.
export function useTender(id: string) {
  const { data, isLoading, error } = useGetTender(id, {
    query: {
      refetchInterval: (query) => {
        const tender = query.state.data
        if (!tender) return false
        const pending = tender.source === "gebiz" && !tender.detailsFetchedAt
        if (!pending || query.state.dataUpdateCount > TENDER_POLL_MAX_ATTEMPTS) return false
        return TENDER_POLL_INTERVAL_MS
      },
    },
  })
  return { data, isLoading, error }
}

// Star/unstar a tender; invalidates tender queries so list/detail views
// pick up the new isFavourite state.
export function useToggleTenderFavourite() {
  const queryClient = useQueryClient()
  const invalidate = () => invalidateTenderQueries(queryClient)
  const add = useAddTenderFavourite({ mutation: { onSuccess: invalidate } })
  const remove = useRemoveTenderFavourite({ mutation: { onSuccess: invalidate } })
  return {
    toggle: (id: string, isFavourite: boolean) =>
      isFavourite ? remove.mutate({ id }) : add.mutate({ id }),
    isPending: add.isPending || remove.isPending,
  }
}

// Triggers AI bid brief generation for a tender; invalidates tender queries on
// success so the detail view picks up the newly-generated brief.
export function useGenerateTenderBrief() {
  const queryClient = useQueryClient()
  return useGenerateTenderBrief_({
    mutation: { onSuccess: () => invalidateTenderQueries(queryClient) },
  })
}

// Triggers AI delivery timeline generation for a tender; invalidates tender queries
// on success so the detail view picks up the newly-generated timeline.
export function useGenerateTenderDeliveryTimeline() {
  const queryClient = useQueryClient()
  return useGenerateTenderDeliveryTimeline_({
    mutation: { onSuccess: () => invalidateTenderQueries(queryClient) },
  })
}

// Triggers AI match-score generation for a tender, scored against the signed-in
// user's company profile; invalidates tender queries on success so the recommended
// list and detail view pick up the newly-generated score.
export function useGenerateTenderMatch() {
  const queryClient = useQueryClient()
  return useGenerateTenderMatch_({
    mutation: { onSuccess: () => invalidateTenderQueries(queryClient) },
  })
}

// Distinct status/source/agency values for filter dropdowns; these change rarely.
// Passing source (pipe-separated) scopes the returned agencies to those sources so
// the Agency dropdown only lists agencies under the selected Source(s).
export function useTenderFilters(source?: string) {
  // keepPreviousData: selecting a source refetches this payload; without it the
  // Source/Status lists would flash empty mid-refetch even though only agencies narrow.
  const { data, isPending } = useGetTenderFilters(
    { source: source || undefined },
    { query: { staleTime: 5 * 60 * 1000, placeholderData: keepPreviousData } },
  )
  return {
    statuses: data?.statuses ?? [],
    sources: data?.sources ?? [],
    agencies: data?.agencies ?? [],
    // false until the first fetch resolves — callers use this to avoid pruning a
    // deep-linked agency selection against an empty list before agencies load.
    loaded: !isPending,
  }
}
