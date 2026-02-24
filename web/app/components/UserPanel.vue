<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'

defineProps<{
  collapsed?: boolean
}>()

// Mock user data - would come from auth in real app
const user = ref({
  name: 'Admin User',
  email: 'admin@pressluft.io',
  avatar: {
    src: undefined,
    alt: 'Admin User'
  }
})

const items = computed<DropdownMenuItem[][]>(() => [[
  {
    type: 'label',
    label: user.value.name,
    description: user.value.email
  }
], [
  {
    label: 'Profile',
    icon: 'i-lucide-user'
  },
  {
    label: 'Settings',
    icon: 'i-lucide-settings',
    to: '/settings'
  }
], [
  {
    label: 'Sign out',
    icon: 'i-lucide-log-out',
    color: 'error'
  }
]])
</script>

<template>
  <UDropdownMenu
    :items="items"
    :content="{ align: 'center', collisionPadding: 12 }"
    :ui="{ content: collapsed ? 'w-48' : 'w-(--reka-dropdown-menu-trigger-width)' }"
  >
    <UButton
      color="neutral"
      variant="ghost"
      block
      :square="collapsed"
      :label="collapsed ? undefined : user?.name"
      :avatar="collapsed ? undefined : user"
      :class="collapsed ? '' : 'justify-start'"
      class="data-[state=open]:bg-neutral-700/50"
    >
      <template #leading>
        <UAvatar v-if="!collapsed" :alt="user.name" size="xs" />
      </template>
      <template #trailing>
        <UIcon v-if="!collapsed" name="i-lucide-chevrons-up-down" class="ml-auto h-4 w-4 text-neutral-400" />
      </template>
    </UButton>
  </UDropdownMenu>
</template>
