<script setup lang="ts">
const props = defineProps<{
  collapsed?: boolean
}>()

const router = useRouter()

// Mock user data - would come from auth in real app
const user = ref({
  name: 'Admin User',
  email: 'admin@pressluft.io',
})

const initials = computed(() =>
  user.value.name
    .split(' ')
    .filter(Boolean)
    .map((part) => part[0])
    .slice(0, 2)
    .join('')
    .toUpperCase(),
)

type MenuIcon = 'user' | 'settings' | 'logout'

interface MenuItem {
  id: string
  label: string
  icon: MenuIcon
  to?: string
  danger?: boolean
}

const menuSections: MenuItem[][] = [
  [
    { id: 'profile', label: 'Profile', icon: 'user' },
    { id: 'settings', label: 'Settings', icon: 'settings', to: '/settings' },
  ],
  [{ id: 'sign-out', label: 'Sign out', icon: 'logout', danger: true }],
]

const handleItemClick = (item: MenuItem) => {
  if (item.to) {
    router.push(item.to)
  }
}

const iconPath = (icon: MenuIcon): string => {
  if (icon === 'settings') {
    return 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 0 0 1.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 0 0-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 0 0-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 0 0-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 0 0-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 0 0 1.066-2.573c-.94-1.543.826-3.31 2.37-2.37a1.724 1.724 0 0 0 2.573-1.066z'
  }

  if (icon === 'logout') {
    return 'M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4m7 14 5-5-5-5m5 5H9'
  }

  return 'M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2'
}
</script>

<template>
  <UiDropdown :align="props.collapsed ? 'right' : 'left'">
    <template #trigger>
      <UiButton
        variant="ghost"
        class="w-full h-full rounded-none px-4 py-3"
        :class="props.collapsed ? 'justify-center' : 'justify-start'"
        type="button"
      >
        <span class="flex items-center gap-3 w-full">
          <span
            class="flex h-8 w-8 items-center justify-center rounded-full bg-surface-800 text-xs font-semibold text-surface-200"
          >
            {{ initials || 'AU' }}
          </span>
          <span v-if="!props.collapsed" class="min-w-0 flex-1 text-left">
            <span class="block text-sm font-medium text-surface-100">{{ user.name }}</span>
            <span class="block truncate text-xs text-surface-500">{{ user.email }}</span>
          </span>
          <svg
            v-if="!props.collapsed"
            class="ml-auto h-4 w-4 text-surface-500"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="M7 15l5 5 5-5" />
            <path stroke-linecap="round" stroke-linejoin="round" d="M7 9l5-5 5 5" />
          </svg>
        </span>
      </UiButton>
    </template>

    <div :class="props.collapsed ? 'w-48' : 'w-56'">
      <div class="px-3 py-2">
        <p class="text-xs font-medium text-surface-200">{{ user.name }}</p>
        <p class="text-xs text-surface-500">{{ user.email }}</p>
      </div>
      <div class="border-t border-surface-800/60" />

      <div class="space-y-1 py-1">
        <template v-for="(section, sectionIndex) in menuSections" :key="sectionIndex">
          <UiDropdownItem
            v-for="item in section"
            :key="item.id"
            :danger="item.danger"
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
            <span>{{ item.label }}</span>
          </UiDropdownItem>
          <div v-if="sectionIndex < menuSections.length - 1" class="border-t border-surface-800/60" />
        </template>
      </div>
    </div>
  </UiDropdown>
</template>
