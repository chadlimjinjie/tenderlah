"use client"

import { useEffect, useRef, useState } from "react"
import { CheckIcon, LayoutGridIcon, SearchIcon, TableIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Drawer,
  DrawerContent,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer"
import { Input } from "@/components/ui/input"
import {
  Item,
  ItemContent,
  ItemDescription,
  ItemTitle,
} from "@/components/ui/item"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { useTenderFilters } from "@/features/tenders/hooks/use-tenders"
import { useIsMobile } from "@/hooks/use-mobile"
import { cn } from "@/lib/utils"

export type TendersView = "table" | "list"
export type SortOption =
  | "relevance:desc"
  | "closingDate:asc"
  | "closingDate:desc"
  | "publishedDate:desc"
  | "publishedDate:asc"
  | "title:asc"
  | "title:desc"

export type OpenState = "" | "open" | "closed"

export interface TendersFilterState {
  q: string
  source: string
  agency: string
  openState: OpenState
  sort: SortOption
  view: TendersView
}

const ALL = "all"

const SORT_OPTIONS: { value: SortOption; label: string }[] = [
  { value: "closingDate:asc", label: "Closing soonest" },
  { value: "closingDate:desc", label: "Closing latest" },
  { value: "publishedDate:desc", label: "Published newest" },
  { value: "publishedDate:asc", label: "Published oldest" },
  { value: "title:asc", label: "Title A–Z" },
  { value: "title:desc", label: "Title Z–A" },
]

export function tendersSortToParams(sort: SortOption) {
  const [sortBy, sortDir] = sort.split(":") as [
    "relevance" | "closingDate" | "publishedDate" | "title",
    "asc" | "desc",
  ]
  return { sortBy, sortDir }
}

interface FilterPickerProps {
  label: string
  value: string
  options: { value: string; label: string }[]
  onSelect: (value: string) => void
  // When set, options[0] is treated as the "clear/all" entry and any number of the
  // remaining options can be selected at once; onSelect receives a pipe-joined value.
  // Pipe (not comma) because agency names can contain commas.
  multiple?: boolean
}

const MULTI_SEP = "|"

function FilterPicker({ label, value, options, onSelect, multiple }: FilterPickerProps) {
  const isMobile = useIsMobile()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState("")

  const changeOpen = (next: boolean) => {
    setOpen(next)
    if (!next) setQuery("")
  }

  // ponytail: only worth a search box on long lists (Agency); Status/Sort stay clean.
  const showSearch = options.length > 8
  const filtered = query
    ? options.filter(
        (o, i) =>
          (multiple && i === 0) || o.label.toLowerCase().includes(query.toLowerCase())
      )
    : options

  const selected = multiple ? new Set(value ? value.split(MULTI_SEP) : []) : new Set([value])
  const isSelected = (v: string) =>
    multiple && v === options[0].value ? selected.size === 0 : selected.has(v)

  let summaryNode: React.ReactNode
  if (!multiple) {
    summaryNode = (options.find((o) => o.value === value) ?? options[0])?.label
  } else {
    const selectedLabels = options.filter((o) => selected.has(o.value)).map((o) => o.label)
    if (selectedLabels.length === 0) {
      summaryNode = options[0].label
    } else if (selectedLabels.length === 1) {
      summaryNode = selectedLabels[0]
    } else {
      summaryNode = (
        <span className="flex min-w-0 items-center">
          <span className="truncate">{selectedLabels[0]}</span>
          <Badge variant="secondary" className="ml-1.5 h-4 min-w-4 px-1 tabular-nums">
            {selectedLabels.length}
          </Badge>
        </span>
      )
    }
  }

  const handleSelect = (v: string) => {
    if (!multiple) {
      onSelect(v)
      changeOpen(false)
      return
    }
    // options[0] = "all": clears the selection.
    if (v === options[0].value) {
      onSelect("")
      return
    }
    const next = new Set(selected)
    if (next.has(v)) next.delete(v)
    else next.add(v)
    onSelect([...next].join(MULTI_SEP))
  }

  const trigger = (
    <Item asChild variant="outline" size="sm" className="w-full min-w-0 cursor-pointer">
      <a type="button" onClick={() => changeOpen(true)}>
        <ItemContent className="min-w-0 gap-0">
          <ItemTitle className="text-xs font-normal text-muted-foreground">{label}</ItemTitle>
          <ItemDescription className="truncate text-foreground">{summaryNode}</ItemDescription>
        </ItemContent>
      </a>
    </Item>
  )

  const panel = (
    <>
      {showSearch && (
        <div className="relative px-2 pb-1">
          <SearchIcon className="pointer-events-none absolute top-1/2 left-5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            autoFocus
            placeholder={`Search ${label.toLowerCase()}`}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="w-full pl-9"
          />
        </div>
      )}
      <div className="flex max-h-[55vh] flex-col gap-1 overflow-y-auto p-2">
        {filtered.map((o) => (
          <button
            key={o.value}
            type="button"
            onClick={() => handleSelect(o.value)}
            className={cn(
              "flex items-center justify-between rounded-md px-3 py-2 text-left text-sm hover:bg-accent",
              isSelected(o.value) && "font-medium"
            )}
          >
            <span className="truncate">{o.label}</span>
            {isSelected(o.value) && <CheckIcon className="size-4 shrink-0" />}
          </button>
        ))}
      </div>
    </>
  )

  if (isMobile) {
    return (
      <>
        {trigger}
        <Drawer open={open} onOpenChange={changeOpen} repositionInputs={false}>
          <DrawerContent className="h-[60vh]">
            <DrawerHeader>
              <DrawerTitle>{label}</DrawerTitle>
            </DrawerHeader>
            {panel}
          </DrawerContent>
        </Drawer>
      </>
    )
  }

  return (
    <>
      {trigger}
      <Dialog open={open} onOpenChange={changeOpen}>
        <DialogContent className="p-0 sm:max-w-sm">
          <DialogHeader className="p-4">
            <DialogTitle className="text-center">{label}</DialogTitle>
          </DialogHeader>
          {panel}
        </DialogContent>
      </Dialog>
    </>
  )
}

interface TendersToolbarProps {
  filters: TendersFilterState
  onChange: (next: Partial<TendersFilterState>) => void
}

export function TendersToolbar({ filters, onChange }: TendersToolbarProps) {
  const { sources, agencies, loaded: agenciesLoaded } = useTenderFilters(filters.source)

  // When a source is unselected the Agency list narrows (server-scoped). Drop any
  // selected agencies that no longer belong to a still-selected source, keeping the
  // rest — rather than clearing the whole Agency filter. Guarded by agenciesLoaded so
  // a deep-linked ?agency= isn't wiped before the list finishes loading.
  useEffect(() => {
    if (!agenciesLoaded || !filters.agency) return
    const available = new Set(agencies)
    const selected = filters.agency.split(MULTI_SEP)
    const kept = selected.filter((a) => available.has(a))
    if (kept.length !== selected.length) onChange({ agency: kept.join(MULTI_SEP) })
    // onChange is recreated each render; excluded to avoid re-running on every render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [agencies, agenciesLoaded, filters.agency])

  // Local copy so typing doesn't push a URL update on every keystroke.
  const [q, setQ] = useState(filters.q)
  const [syncedQ, setSyncedQ] = useState(filters.q)
  if (filters.q !== syncedQ) {
    setSyncedQ(filters.q)
    setQ(filters.q)
  }

  const debounceRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  const handleSearchChange = (value: string) => {
    setQ(value)
    clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => onChange({ q: value }), 300)
  }

  return (
    <Card
      className={cn(
        "mb-4 mx-auto w-full transition-[max-width] duration-300 ease-out",
        filters.view === "list" ? "max-w-3xl" : "max-w-7xl"
      )}
    >
      <CardContent className="flex min-w-0 flex-col gap-4">
        <div className="flex items-center gap-3">
          <div className="relative min-w-0 flex-1">
            <SearchIcon className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search tenders, agencies or IDs"
              value={q}
              onChange={(e) => handleSearchChange(e.target.value)}
              className="w-full pl-9"
            />
          </div>
          <ToggleGroup
            type="single"
            variant="outline"
            value={filters.view}
            onValueChange={(v) => v && onChange({ view: v as TendersView })}
            className="shrink-0"
          >
            <ToggleGroupItem value="list" aria-label="List view">
              <LayoutGridIcon className="size-4" />
              <span className="hidden sm:inline">List</span>
            </ToggleGroupItem>
            <ToggleGroupItem value="table" aria-label="Table view">
              <TableIcon className="size-4" />
              <span className="hidden sm:inline">Table</span>
            </ToggleGroupItem>
          </ToggleGroup>
        </div>

        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <FilterPicker
            label="Source"
            multiple
            value={filters.source}
            options={[
              { value: ALL, label: "All sources" },
              ...sources.map((s) => ({ value: s, label: s.toUpperCase() })),
            ]}
            onSelect={(v) => onChange({ source: v })}
          />
          <FilterPicker
            label="Status"
            value={filters.openState || ALL}
            options={[
              { value: ALL, label: "All" },
              { value: "open", label: "Open" },
              { value: "closed", label: "Closed" },
            ]}
            onSelect={(v) => onChange({ openState: v === ALL ? "" : (v as OpenState) })}
          />
          <FilterPicker
            label="Agency"
            multiple
            value={filters.agency}
            options={[
              { value: ALL, label: "All agencies" },
              ...agencies.map((a) => ({ value: a, label: a })),
            ]}
            onSelect={(v) => onChange({ agency: v })}
          />
          <FilterPicker
            label="Sort by"
            value={filters.sort}
            options={SORT_OPTIONS}
            onSelect={(v) => onChange({ sort: v as SortOption })}
          />
        </div>
      </CardContent>
    </Card>
  )
}
