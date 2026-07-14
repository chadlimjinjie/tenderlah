"use client"

import { useEffect, useState } from "react"
import Link from "next/link"
import { usePathname } from "next/navigation"
import { ChevronRightIcon } from "lucide-react"

import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import {
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
} from "@/components/ui/sidebar"

export type NavGroupItem = {
  title: string
  url: string
  icon?: React.ReactNode
}

// Module-level cache survives remounts when navigating between route groups.
const openStateCache = new Map<string, boolean>()

/**
 * A collapsible parent entry for the sidebar menu. Renders a single
 * `SidebarMenuItem`, so it can sit inside any existing `SidebarMenu`
 * alongside flat menu items.
 */
export function NavGroup({
  label,
  icon,
  items,
}: {
  label: string
  icon?: React.ReactNode
  items: NavGroupItem[]
}) {
  const pathname = usePathname()
  const isRouteActive = (url: string) =>
    pathname === url || pathname.startsWith(`${url}/`)
  const anyActive = items.some((item) => isRouteActive(item.url))
  const [open, setOpen] = useState<boolean>(
    () => openStateCache.get(label) ?? anyActive
  )

  const handleOpenChange = (value: boolean) => {
    openStateCache.set(label, value)
    setOpen(value)
  }

  useEffect(() => {
    if (anyActive) {
      openStateCache.set(label, true)
      setOpen(true)
    }
  }, [anyActive, label])

  return (
    <Collapsible
      open={open}
      onOpenChange={handleOpenChange}
      className="group/collapsible"
    >
      <SidebarMenuItem>
        <CollapsibleTrigger asChild>
          <SidebarMenuButton isActive={anyActive} tooltip={label}>
            {icon}
            <span>{label}</span>
            <ChevronRightIcon className="ms-auto transition-transform group-data-[state=open]/collapsible:rotate-90" />
          </SidebarMenuButton>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <SidebarMenuSub>
            {items.map((item) => (
              <SidebarMenuSubItem key={item.url}>
                <SidebarMenuSubButton asChild isActive={isRouteActive(item.url)}>
                  <Link href={item.url}>
                    {item.icon}
                    <span>{item.title}</span>
                  </Link>
                </SidebarMenuSubButton>
              </SidebarMenuSubItem>
            ))}
          </SidebarMenuSub>
        </CollapsibleContent>
      </SidebarMenuItem>
    </Collapsible>
  )
}
