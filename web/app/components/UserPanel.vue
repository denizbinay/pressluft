<script setup lang="ts">
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Button } from "@/components/ui/button"
import { useSidebar } from "@/components/ui/sidebar"
import { cn } from "@/lib/utils"

const props = defineProps<{
  collapsed?: boolean
}>()

const router = useRouter()
const colorMode = useColorMode()
const { state } = useSidebar()

// Mock user data - would come from auth in real app
const user = ref({
  name: "Admin User",
  email: "admin@pressluft.io",
})

const initials = computed(() =>
  user.value.name
    .split(" ")
    .filter(Boolean)
    .map((part) => part[0])
    .slice(0, 2)
    .join("")
    .toUpperCase(),
)

const isCollapsed = computed(() => props.collapsed ?? state.value === "collapsed")

type MenuIcon = "user" | "settings" | "theme" | "logout"

interface MenuItem {
  id: string
  label: string
  icon: MenuIcon
  to?: string
  danger?: boolean
}

const menuSections: MenuItem[][] = [
  [
    { id: "profile", label: "Profile", icon: "user" },
    { id: "settings", label: "Settings", icon: "settings", to: "/settings" },
  ],
  [{ id: "theme", label: "Theme", icon: "theme" }],
  [{ id: "sign-out", label: "Sign out", icon: "logout", danger: true }],
]

const formatModeLabel = (mode: string) => `${mode.charAt(0).toUpperCase()}${mode.slice(1)}`

const themeLabel = computed(() => {
  const current = colorMode.preference ?? "system"
  return `Theme: ${formatModeLabel(current)}`
})

const toggleThemePreference = () => {
  const current = colorMode.preference ?? "system"
  const next = current === "system" ? "light" : current === "light" ? "dark" : "system"
  colorMode.preference = next
}

const handleItemClick = (item: MenuItem) => {
  if (item.id === "theme") {
    toggleThemePreference()
    return
  }

  if (item.to) {
    router.push(item.to)
  }
}

const iconPath = (icon: MenuIcon): string => {
  if (icon === "settings") {
    return "M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 0 0 1.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 0 0-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 0 0-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 0 0-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 0 0-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 0 0 1.066-2.573c-.94-1.543.826-3.31 2.37-2.37a1.724 1.724 0 0 0 2.573-1.066z"
  }

  if (icon === "theme") {
    return "M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"
  }

  if (icon === "logout") {
    return "M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4m7 14 5-5-5-5m5 5H9"
  }

  return "M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"
}
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button
        variant="ghost"
        :class="cn(
          'w-full h-full rounded-none px-4 py-3 hover:bg-muted/60 hover:text-foreground focus-visible:bg-muted/60',
          isCollapsed ? 'justify-center' : 'justify-start',
        )"
        type="button"
      >
        <span class="flex items-center gap-3 w-full">
          <span
            class="flex h-8 w-8 items-center justify-center rounded-full bg-muted text-xs font-semibold text-foreground"
          >
            {{ initials || "AU" }}
          </span>
          <span v-if="!isCollapsed" class="min-w-0 flex-1 text-left">
            <span class="block text-sm font-medium text-foreground">{{ user.name }}</span>
            <span class="block truncate text-xs text-muted-foreground">{{ user.email }}</span>
          </span>
          <svg
            v-if="!isCollapsed"
            class="ml-auto h-4 w-4 text-muted-foreground"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="M7 15l5 5 5-5" />
            <path stroke-linecap="round" stroke-linejoin="round" d="M7 9l5-5 5 5" />
          </svg>
        </span>
      </Button>
    </DropdownMenuTrigger>

    <DropdownMenuContent
      :align="isCollapsed ? 'end' : 'start'"
      :side-offset="6"
      :class="cn('rounded-lg border border-border/60 bg-popover p-1 shadow-xl text-popover-foreground')"
    >
      <div :class="cn(isCollapsed ? 'w-48' : 'w-56')">
        <div class="px-3 py-2">
          <p class="text-xs font-medium text-foreground">{{ user.name }}</p>
          <p class="text-xs text-muted-foreground">{{ user.email }}</p>
        </div>
        <DropdownMenuSeparator class="mx-0 my-0 h-px bg-border/60" />

        <div class="space-y-1 py-1">
          <template v-for="(section, sectionIndex) in menuSections" :key="sectionIndex">
            <DropdownMenuItem
              v-for="item in section"
              :key="item.id"
              :class="cn(
                'flex w-full items-center gap-2 rounded-md px-3 py-1.5 text-left text-sm transition-colors',
                item.danger
                  ? 'text-destructive hover:bg-destructive/10 hover:text-destructive focus:bg-destructive/10 focus:text-destructive'
                  : item.id === 'theme'
                    ? 'text-foreground hover:bg-muted/60 focus:bg-muted/60'
                    : 'text-muted-foreground hover:bg-muted/60 focus:bg-muted/60',
              )"
              @click="handleItemClick(item)"
            >
              <svg
                class="h-4 w-4"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <path stroke-linecap="round" stroke-linejoin="round" :d="iconPath(item.icon)" />
                <circle v-if="item.icon === 'user'" cx="12" cy="7" r="4" />
              </svg>
              <span>{{ item.id === 'theme' ? themeLabel : item.label }}</span>
            </DropdownMenuItem>
            <DropdownMenuSeparator
              v-if="sectionIndex < menuSections.length - 1"
              class="mx-0 my-0 h-px bg-border/60"
            />
          </template>
        </div>
      </div>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
