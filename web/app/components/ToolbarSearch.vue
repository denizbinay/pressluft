<script setup lang="ts">
import type { Component } from "vue"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Search } from "lucide-vue-next"
import { computed, onBeforeUnmount, onMounted, watch } from "vue"

interface ToolbarSearchItem {
  label: string
  to: string
  icon: Component
}

const props = defineProps<{
  items: ToolbarSearchItem[]
}>()

const route = useRoute()
const router = useRouter()
const searchOpen = ref(false)
const searchQuery = ref("")

const filteredItems = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  if (!query) return props.items
  return props.items.filter((item) => item.label.toLowerCase().includes(query))
})

const openSearch = () => {
  searchOpen.value = true
}

const closeSearch = () => {
  searchOpen.value = false
  searchQuery.value = ""
}

const handleSearchSelect = (item: ToolbarSearchItem) => {
  router.push(item.to)
  closeSearch()
}

const handleKeydown = (event: KeyboardEvent) => {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
    event.preventDefault()
    openSearch()
  }
}

watch(
  () => route.fullPath,
  () => {
    closeSearch()
  },
)

onMounted(() => {
  window.addEventListener("keydown", handleKeydown)
})

onBeforeUnmount(() => {
  window.removeEventListener("keydown", handleKeydown)
})
</script>

<template>
  <div class="contents">
    <Button
      variant="ghost"
      class="hidden sm:flex rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted/70 focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
      type="button"
      @click="openSearch"
    >
      <Search class="h-4 w-4" aria-hidden="true" />
      <span>Search...</span>
      <span class="ml-2 rounded border border-border/60 px-1.5 py-0.5 text-xs text-muted-foreground">⌘K</span>
    </Button>

    <Button
      variant="ghost"
      class="sm:hidden rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted/70 focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
      type="button"
      aria-label="Open search"
      @click="openSearch"
    >
      <Search class="h-4 w-4" aria-hidden="true" />
    </Button>

    <Dialog :open="searchOpen" @update:open="(value) => (value ? openSearch() : closeSearch())">
      <DialogContent
        :show-close-button="false"
        class="w-full max-w-lg rounded-xl border border-border/60 bg-popover/90 p-0 text-popover-foreground shadow-2xl"
      >
        <div class="flex items-center justify-between border-b border-border/40 px-6 py-4">
          <DialogTitle class="text-base font-semibold text-foreground">Search</DialogTitle>
          <DialogClose
            class="inline-flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground transition hover:bg-muted/60 hover:text-foreground"
            aria-label="Close modal"
          >
            <span aria-hidden="true">x</span>
          </DialogClose>
        </div>

        <div class="space-y-4 px-6 py-5">
          <Input
            v-model="searchQuery"
            type="search"
            placeholder="Search pages..."
            autofocus
            class="w-full rounded-lg border border-border/60 bg-background/60 px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground transition-colors hover:border-border focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
          />

          <div class="space-y-1">
            <p class="text-xs font-medium uppercase tracking-wider text-muted-foreground">Navigation</p>
            <div v-if="filteredItems.length" class="space-y-1">
              <button
                v-for="item in filteredItems"
                :key="item.label"
                type="button"
                class="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left text-sm text-foreground/80 transition-colors hover:bg-muted/60"
                @click="handleSearchSelect(item)"
              >
                <component :is="item.icon" class="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" />
                <span>{{ item.label }}</span>
                <span class="ml-auto text-xs text-muted-foreground">{{ item.to }}</span>
              </button>
            </div>
            <p v-else class="text-sm text-muted-foreground">No matches found.</p>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>
