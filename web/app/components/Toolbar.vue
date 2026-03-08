<script setup lang="ts">
import type { Component } from "vue"

interface ToolbarItem {
  label: string
  to: string
  icon: Component
}

defineProps<{
  items: ToolbarItem[]
}>()
</script>

<template>
  <div class="flex items-center gap-2">
    <ToolbarSearch :items="items" />
    <ToolbarHelpButton />
    <ToolbarNotifications />
  </div>
</template>
