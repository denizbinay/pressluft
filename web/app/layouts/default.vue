<script setup lang="ts">
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar"
import {
  Bell,
  Box,
  Globe,
  HelpCircle,
  LayoutDashboard,
  Search,
  Server,
  Settings,
} from 'lucide-vue-next'
import { computed, onBeforeUnmount, onMounted, watch } from 'vue'

const route = useRoute()
const router = useRouter()

const searchOpen = ref(false)
const searchQuery = ref('')

const navSections = [
  {
    title: 'Main',
    items: [
      { label: 'Dashboard', icon: LayoutDashboard, to: '/' },
    ],
  },
  {
    title: 'Resources',
    items: [
      { label: 'Sites', icon: Globe, to: '/sites' },
      { label: 'Servers', icon: Server, to: '/servers' },
    ],
  },
  {
    title: 'System',
    items: [
      { label: 'Settings', icon: Settings, to: '/settings' },
      { label: 'Components', icon: Box, to: '/components' },
    ],
  },
]

const flatNavItems = navSections.flatMap((section) => section.items)

const filteredNavItems = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  if (!query) return flatNavItems
  return flatNavItems.filter((item) =>
    item.label.toLowerCase().includes(query),
  )
})

const breadcrumbLabel = computed(() => {
  if (route.path === '/') return 'Dashboard'
  return route.path.replace('/', '').replace('-', ' ')
})

const isActive = (to: string) =>
  route.path === to || route.path.startsWith(`${to}/`)

const openSearch = () => {
  searchOpen.value = true
}

const closeSearch = () => {
  searchOpen.value = false
  searchQuery.value = ''
}

const handleSearchSelect = (item: { to: string }) => {
  router.push(item.to)
  closeSearch()
}

const handleKeydown = (event: KeyboardEvent) => {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
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
  window.addEventListener('keydown', handleKeydown)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <SidebarProvider class="min-h-screen bg-background text-foreground">
    <Sidebar
      collapsible="icon"
      class="border-r border-border/60 bg-background"
    >
      <SidebarHeader class="flex px-4 py-4">
        <NuxtLink to="/" class="flex items-center gap-2">
          <div class="flex h-9 w-9 items-center justify-center rounded-lg text-accent">
            <svg
              version="1.1"
              xmlns="http://www.w3.org/2000/svg"
              xmlns:xlink="http://www.w3.org/1999/xlink"
              viewBox="390 220 250 220"
              xml:space="preserve"
              class="h-7 w-7"
            >
              <defs>
                <linearGradient id="pl-grad" x1="280" y1="210" x2="740" y2="460" gradientUnits="userSpaceOnUse">
                  <stop offset="0%" stop-color="#1F5B66" />
                  <stop offset="45%" stop-color="#2A7A86" />
                  <stop offset="100%" stop-color="#45C6D6" />
                </linearGradient>
                <linearGradient id="pl-grad-d" x1="260" y1="240" x2="720" y2="500" gradientUnits="userSpaceOnUse">
                  <stop offset="0%" stop-color="#173F48" />
                  <stop offset="55%" stop-color="#226772" />
                  <stop offset="100%" stop-color="#38AFC0" />
                </linearGradient>
                <filter id="pl-glow" x="-20%" y="-20%" width="140%" height="140%">
                  <feDropShadow dx="0" dy="10" stdDeviation="14" flood-color="#2AA9B8" flood-opacity="0.14" />
                  <feDropShadow dx="0" dy="2" stdDeviation="4" flood-color="#000000" flood-opacity="0.35" />
                </filter>
              </defs>
              <g filter="url(#pl-glow)">
                <path fill="url(#pl-grad)" stroke="none" d=" M579.609558,298.002838 C579.611511,293.005035 579.719299,288.503815 579.598083,284.008728 C579.309204,273.302124 573.351624,267.415009 562.582764,267.338928 C548.421509,267.238861 534.259155,267.298492 520.097229,267.298370 C498.104645,267.298157 476.112030,267.310242 454.119446,267.306946 C446.798248,267.305847 446.792084,267.291504 446.790924,259.764618 C446.789856,252.933578 446.755585,246.102081 446.822601,239.271622 C446.933868,227.929245 445.546692,228.951553 457.427338,228.942734 C503.411804,228.908600 549.396301,228.934998 595.380798,228.940964 C600.878906,228.941681 606.380798,228.827377 611.874329,228.989960 C622.729797,229.311295 629.285217,235.936996 629.310486,246.672302 C629.364624,269.664429 629.218567,292.657532 629.393433,315.648407 C629.426697,320.025421 628.080750,321.448395 623.686646,321.363861 C610.862305,321.117249 598.027649,321.124023 585.202637,321.350342 C580.736816,321.429108 579.344604,319.775574 579.556763,315.492981 C579.836731,309.842346 579.616455,304.166962 579.609558,298.002838 z" />
                <path fill="url(#pl-grad-d)" stroke="none" d=" M445.000000,374.194580 C434.345367,374.188812 424.190430,374.229126 414.036194,374.163666 C402.047974,374.086426 396.663483,368.671783 396.663177,356.812897 C396.662506,331.176605 396.771301,305.539520 396.580750,279.904663 C396.545258,275.129791 397.971741,273.446075 402.818726,273.653534 C410.961029,274.001984 419.135010,273.953949 427.283112,273.688232 C431.773132,273.541870 433.060120,275.413086 433.009491,279.589294 C432.851990,292.571991 432.863556,305.558228 432.954865,318.542053 C433.044922,331.347595 437.433594,335.663208 450.118164,335.684235 C462.436859,335.704651 474.755646,335.687805 487.074371,335.693817 C488.572021,335.694519 490.078156,335.640564 491.565918,335.773499 C499.448303,336.477905 500.320953,337.416260 500.347565,345.248230 C500.374115,353.072174 500.183350,360.901276 500.416473,368.718323 C500.539337,372.837402 499.040619,374.276093 494.941010,374.239990 C478.461792,374.094788 461.980530,374.188507 445.000000,374.194580 z" />
                <path fill="url(#pl-grad-d)" stroke="none" d=" M569.000000,425.321045 C538.516846,425.316986 508.533661,425.320312 478.550476,425.301270 C471.973297,425.297119 471.886780,425.185669 471.884338,418.559753 C471.881317,410.397736 472.028442,402.233093 471.859680,394.074799 C471.773895,389.929047 473.139587,388.007385 477.564606,388.015961 C514.876587,388.088226 552.188721,388.045746 589.500793,387.989471 C593.165894,387.983948 594.815308,389.040894 594.670837,393.233917 C594.384583,401.542053 594.848755,409.873627 594.883667,418.196045 C594.912781,425.137390 594.685181,425.304779 587.989807,425.319183 C581.826538,425.332428 575.663269,425.321014 569.000000,425.321045 z" />
                <path fill="url(#pl-grad)" stroke="none" d=" M472.000092,282.676422 C496.824921,282.678497 521.149780,282.665100 545.474609,282.689850 C556.539612,282.701080 561.298950,287.470245 561.330872,298.493073 C561.348267,304.490906 561.204712,310.492310 561.371460,316.485443 C561.465393,319.861694 560.293823,321.275543 556.790344,321.268921 C521.969360,321.203339 487.148132,321.210297 452.327179,321.281067 C448.890533,321.288025 447.577362,319.958191 447.615570,316.549438 C447.725800,306.720856 447.719269,296.889313 447.600708,287.060822 C447.561951,283.846710 448.802185,282.583466 452.007721,282.645599 C458.502960,282.771484 465.002380,282.679016 472.000092,282.676422 z" />
                <path fill="url(#pl-grad)" stroke="none" d=" M581.000000,335.735168 C595.149536,335.737549 608.799133,335.734253 622.448669,335.742828 C629.267334,335.747101 629.292114,335.756836 629.306213,342.765625 C629.323975,351.587830 629.180298,360.412262 629.343323,369.231323 C629.411011,372.892242 628.051147,374.243805 624.400879,374.233765 C594.938049,374.152618 565.474792,374.148560 536.011963,374.234802 C532.315796,374.245605 530.947815,372.740997 530.989807,369.192505 C531.100220,359.872040 531.173401,350.548523 531.050659,341.229034 C530.993591,336.890747 532.782471,335.564514 537.055603,335.632629 C551.533997,335.863495 566.018066,335.729156 581.000000,335.735168 z" />
                <path fill="url(#pl-grad-d)" stroke="none" d=" M450.587372,388.063721 C454.087769,388.125610 455.014282,389.955444 455.001984,392.672119 C454.959778,402.003876 454.922089,411.335846 454.941071,420.667603 C454.947815,423.987732 453.551056,425.405029 450.100830,425.369354 C437.936981,425.243591 425.768890,425.461487 413.606537,425.276367 C403.700134,425.125610 396.821960,417.949127 396.681763,408.009521 C396.613647,403.177795 396.895630,398.331238 396.611511,393.515411 C396.362335,389.291626 398.154388,387.958832 402.138306,387.986725 C418.135437,388.098785 434.133789,388.039307 450.587372,388.063721 z" />
              </g>
            </svg>
          </div>
          <span class="text-lg font-semibold tracking-tight text-foreground group-data-[collapsible=icon]:hidden">
            Pressluft
          </span>
        </NuxtLink>

        <SidebarTrigger
          class="lg:hidden rounded-md text-muted-foreground transition hover:bg-muted/60 hover:text-foreground"
        />
      </SidebarHeader>

      <SidebarContent class="px-3 py-6 gap-8">
        <div v-for="section in navSections" :key="section.title" class="space-y-3">
          <p
            class="px-3 text-[10px] font-medium uppercase tracking-wider text-muted-foreground group-data-[collapsible=icon]:hidden"
          >
            {{ section.title }}
          </p>
          <div class="space-y-1">
            <NuxtLink
              v-for="item in section.items"
              :key="item.label"
              :to="item.to"
              class="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors group-data-[collapsible=icon]:justify-center"
              :class="[
                isActive(item.to)
                  ? 'bg-muted/60 text-foreground'
                  : 'text-muted-foreground hover:bg-muted/40 hover:text-foreground',
              ]"
              :aria-label="item.label"
              :title="item.label"
            >
              <component :is="item.icon" class="h-4 w-4 shrink-0" aria-hidden="true" />
              <span class="group-data-[collapsible=icon]:hidden">{{ item.label }}</span>
            </NuxtLink>
          </div>
        </div>
      </SidebarContent>

      <SidebarFooter class="border-t border-border/60 p-0">
        <UserPanel />
      </SidebarFooter>
    </Sidebar>

    <SidebarInset class="bg-background text-foreground">
      <header class="flex items-center justify-between border-b border-border/60 bg-background/80 px-4 py-3 backdrop-blur lg:px-6">
        <div class="flex items-center gap-3">
          <SidebarTrigger
            class="lg:hidden rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted/70 focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
          />

          <SidebarTrigger
            class="hidden lg:flex rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted/70 focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
          />

          <div class="flex items-center gap-1.5 text-xs text-muted-foreground">
            <span>Pressluft</span>
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
            <span class="text-foreground capitalize">{{ breadcrumbLabel }}</span>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <Button
            variant="ghost"
            class="hidden sm:flex rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted/70 focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
            type="button"
            @click="openSearch"
          >
             <Search class="h-4 w-4" aria-hidden="true" />
            <span>Search...</span>
            <span class="ml-2 text-xs text-muted-foreground border border-border/60 rounded px-1.5 py-0.5">⌘K</span>
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

          <Button
            variant="ghost"
            class="rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted/70 focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
            type="button"
            aria-label="Help"
          >
             <HelpCircle class="h-4 w-4" aria-hidden="true" />
          </Button>

          <Button
            variant="ghost"
            class="rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted/70 focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
            type="button"
            aria-label="Notifications"
          >
             <Bell class="h-4 w-4" aria-hidden="true" />
          </Button>
        </div>
      </header>

      <main class="flex-1 overflow-auto">
        <div class="h-full p-6 sm:p-8 lg:p-10 xl:p-12">
          <div class="mx-auto max-w-7xl">
            <slot />
          </div>
        </div>
      </main>
    </SidebarInset>

    <Dialog :open="searchOpen" @update:open="(value) => (value ? openSearch() : closeSearch())">
      <DialogContent
        :show-close-button="false"
        class="w-full max-w-lg rounded-xl border border-border/60 bg-popover/90 p-0 shadow-2xl text-popover-foreground"
      >
        <div class="flex items-center justify-between border-b border-border/40 px-6 py-4">
          <DialogTitle class="text-base font-semibold text-foreground">Search</DialogTitle>
          <DialogClose
            class="inline-flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground transition hover:bg-muted/60 hover:text-foreground"
            aria-label="Close modal"
          >
            <span aria-hidden="true">×</span>
          </DialogClose>
        </div>

        <div class="space-y-4 px-6 py-5">
          <Input
            v-model="searchQuery"
            type="search"
            placeholder="Search pages..."
            autofocus
            class="w-full rounded-lg border border-border/60 bg-background/60 px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background hover:border-border"
          />

          <div class="space-y-1">
            <p class="text-xs font-medium uppercase tracking-wider text-muted-foreground">Navigation</p>
            <div v-if="filteredNavItems.length" class="space-y-1">
              <button
                v-for="item in filteredNavItems"
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
  </SidebarProvider>
</template>
