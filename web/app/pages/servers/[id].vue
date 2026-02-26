<script setup lang="ts">
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { useServers, type StoredServer } from "~/composables/useServers"
import { useJobs } from "~/composables/useJobs"

interface ServerSection {
  key: string
  label: string
  icon: string
  description: string
}

const sections: ServerSection[] = [
  { key: 'overview', label: 'Overview', icon: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6', description: 'Server status and quick actions' },
  { key: 'sites', label: 'Sites', icon: 'M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9', description: 'Websites hosted on this server' },
  { key: 'settings', label: 'Settings', icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z', description: 'Server configuration and management' },
  { key: 'activity', label: 'Activity', icon: 'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z', description: 'Recent events and job history' },
]

const route = useRoute()
const router = useRouter()

const serverId = computed(() => {
  const id = Number(route.params.id)
  return Number.isNaN(id) ? null : id
})

const { fetchServer } = useServers()
const { createJob } = useJobs()

const server = ref<StoredServer | null>(null)
const loading = ref(true)
const error = ref('')

const activeSection = computed(() => {
  const tab = route.query.tab as string
  const isValid = sections.some((s) => s.key === tab)
  return isValid ? tab : 'overview'
})

const currentSection = computed(() =>
  sections.find((s) => s.key === activeSection.value)!,
)

const navigateTo = (key: string) => {
  router.push({ query: { tab: key } })
}

const isMobileSidebarOpen = ref(false)

const toggleMobileSidebar = () => {
  isMobileSidebarOpen.value = !isMobileSidebarOpen.value
}

const selectSection = (key: string) => {
  navigateTo(key)
  isMobileSidebarOpen.value = false
}

const statusVariant = (status: string): 'success' | 'warning' | 'danger' | 'default' => {
  if (status === 'ready') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'provisioning' || status === 'pending') return 'warning'
  return 'default'
}

const rebuildServerName = ref("")
const rebuildServerImage = ref("")
const resizeServerType = ref("")
const resizeUpgradeDisk = ref(false)
const updateFirewallsList = ref("")
const volumeName = ref("")
const volumeSizeGb = ref("")
const volumeLocation = ref("")
const volumeState = ref("present")
const volumeAutomount = ref(false)

const rebuildState = reactive({ loading: false, error: "", success: "" })
const resizeState = reactive({ loading: false, error: "", success: "" })
const firewallsState = reactive({ loading: false, error: "", success: "" })
const volumeStateUi = reactive({ loading: false, error: "", success: "" })

const normalizeText = (value: string) => value.trim()

const submitRebuild = async () => {
  if (!serverId.value) return
  rebuildState.loading = true
  rebuildState.error = ""
  rebuildState.success = ""
  try {
    const serverName = normalizeText(rebuildServerName.value)
    const serverImage = normalizeText(rebuildServerImage.value)
    if (!serverName || !serverImage) {
      throw new Error("Server name and image are required")
    }
    const job = await createJob({
      kind: "rebuild_server",
      server_id: serverId.value,
      payload: {
        server_name: serverName,
        server_image: serverImage,
      },
    })
    rebuildState.success = `Job #${job.id} created`
  } catch (e: any) {
    rebuildState.error = e.message || "Failed to create rebuild job"
  } finally {
    rebuildState.loading = false
  }
}

const submitResize = async () => {
  if (!serverId.value) return
  resizeState.loading = true
  resizeState.error = ""
  resizeState.success = ""
  try {
    const serverType = normalizeText(resizeServerType.value)
    if (!serverType) {
      throw new Error("Server type is required")
    }
    const job = await createJob({
      kind: "resize_server",
      server_id: serverId.value,
      payload: {
        server_type: serverType,
        upgrade_disk: resizeUpgradeDisk.value,
      },
    })
    resizeState.success = `Job #${job.id} created`
  } catch (e: any) {
    resizeState.error = e.message || "Failed to create resize job"
  } finally {
    resizeState.loading = false
  }
}

const submitUpdateFirewalls = async () => {
  if (!serverId.value) return
  firewallsState.loading = true
  firewallsState.error = ""
  firewallsState.success = ""
  try {
    const firewalls = updateFirewallsList.value
      .split(",")
      .map((item) => item.trim())
      .filter(Boolean)
    if (firewalls.length === 0) {
      throw new Error("At least one firewall is required")
    }
    const job = await createJob({
      kind: "update_firewalls",
      server_id: serverId.value,
      payload: { firewalls },
    })
    firewallsState.success = `Job #${job.id} created`
  } catch (e: any) {
    firewallsState.error = e.message || "Failed to create firewall update job"
  } finally {
    firewallsState.loading = false
  }
}

const submitManageVolume = async () => {
  if (!serverId.value) return
  volumeStateUi.loading = true
  volumeStateUi.error = ""
  volumeStateUi.success = ""
  try {
    const name = normalizeText(volumeName.value)
    const state = normalizeText(volumeState.value)
    const location = normalizeText(volumeLocation.value)
    const sizeGb = Number.parseInt(volumeSizeGb.value, 10)
    if (!name || !state) {
      throw new Error("Volume name and state are required")
    }
    if (state === "present") {
      if (!location) {
        throw new Error("Location is required when state is present")
      }
      if (!Number.isFinite(sizeGb) || sizeGb <= 0) {
        throw new Error("Size must be a positive number")
      }
    }
    const payload: Record<string, unknown> = {
      volume_name: name,
      state,
    }
    if (state === "present") {
      payload.location = location
      payload.size_gb = sizeGb
      payload.automount = volumeAutomount.value
    }
    const job = await createJob({
      kind: "manage_volume",
      server_id: serverId.value,
      payload,
    })
    volumeStateUi.success = `Job #${job.id} created`
  } catch (e: any) {
    volumeStateUi.error = e.message || "Failed to create volume job"
  } finally {
    volumeStateUi.loading = false
  }
}

watch(server, (value) => {
  if (!value) return
  if (!rebuildServerName.value) rebuildServerName.value = value.name || ""
  if (!rebuildServerImage.value) rebuildServerImage.value = value.image || ""
  if (!resizeServerType.value) resizeServerType.value = value.server_type || ""
  if (!volumeLocation.value) volumeLocation.value = value.location || ""
})

const formatDate = (iso: string): string => {
  try {
    return new Date(iso).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  } catch {
    return iso
  }
}

onMounted(async () => {
  if (!serverId.value) {
    error.value = 'Invalid server ID'
    loading.value = false
    return
  }

  try {
    server.value = await fetchServer(serverId.value)
  } catch (e: any) {
    error.value = e.message || 'Failed to load server'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div>
    <!-- Loading state -->
    <div v-if="loading" class="flex items-center justify-center py-20">
      <svg class="h-6 w-6 animate-spin text-muted-foreground" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
      </svg>
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="space-y-4">
      <div>
        <h1 class="text-2xl font-semibold text-foreground">Server Not Found</h1>
        <p class="mt-1 text-sm text-muted-foreground">{{ error }}</p>
      </div>
      <NuxtLink
        to="/servers"
        class="inline-flex items-center gap-1 text-sm text-accent transition-colors hover:text-accent/80"
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
        </svg>
        Back to Servers
      </NuxtLink>
    </div>

    <!-- Server detail -->
    <template v-else-if="server">
      <!-- Page header -->
      <div class="mb-6">
        <div class="flex items-center gap-2 text-sm text-muted-foreground">
          <NuxtLink to="/servers" class="hover:text-foreground/80 transition-colors">Servers</NuxtLink>
          <span>/</span>
          <span class="text-foreground/80">{{ server.name }}</span>
        </div>
        <div class="mt-2 flex items-center gap-3">
          <h1 class="text-2xl font-semibold text-foreground">{{ server.name }}</h1>
          <Badge
            variant="outline"
            :class="[
              'px-2.5 py-1 text-sm border',
              statusVariant(server.status) === 'success' && 'border-primary/30 bg-primary/10 text-primary',
              statusVariant(server.status) === 'warning' && 'border-accent/30 bg-accent/10 text-accent',
              statusVariant(server.status) === 'danger' && 'border-destructive/30 bg-destructive/10 text-destructive',
              statusVariant(server.status) === 'default' && 'border-border/60 bg-muted/60 text-foreground',
            ]"
          >
            {{ server.status }}
          </Badge>
        </div>
        <p class="mt-1 text-sm text-muted-foreground">
          {{ server.location }} · {{ server.server_type }} · {{ server.profile_key }}
        </p>
      </div>

      <!-- Mobile section selector -->
      <div class="mb-4 lg:hidden">
        <button
          class="flex w-full items-center justify-between rounded-lg border border-border/60 bg-card/50 px-4 py-3 text-sm font-medium text-foreground transition-colors hover:bg-card/70"
          @click="toggleMobileSidebar"
        >
          <span class="flex items-center gap-2.5">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              class="h-4 w-4 text-muted-foreground"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                :d="currentSection.icon"
              />
            </svg>
            {{ currentSection.label }}
          </span>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            class="h-4 w-4 text-muted-foreground transition-transform"
            :class="{ 'rotate-180': isMobileSidebarOpen }"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        <!-- Mobile dropdown -->
        <Transition
          enter-active-class="transition duration-150 ease-out"
          enter-from-class="opacity-0 -translate-y-1"
          enter-to-class="opacity-100 translate-y-0"
          leave-active-class="transition duration-100 ease-in"
          leave-from-class="opacity-100 translate-y-0"
          leave-to-class="opacity-0 -translate-y-1"
        >
          <div
            v-if="isMobileSidebarOpen"
            class="mt-1 overflow-hidden rounded-lg border border-border/60 bg-card/80 backdrop-blur-sm"
          >
            <nav aria-label="Server sections">
              <button
                v-for="section in sections"
                :key="section.key"
                :class="[
                  'flex w-full items-center gap-2.5 px-4 py-2.5 text-left text-sm transition-colors',
                  activeSection === section.key
                    ? 'bg-accent/10 text-accent'
                    : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground',
                ]"
                @click="selectSection(section.key)"
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  class="h-4 w-4 shrink-0"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path stroke-linecap="round" stroke-linejoin="round" :d="section.icon" />
                </svg>
                {{ section.label }}
              </button>
            </nav>
          </div>
        </Transition>
      </div>

      <!-- Desktop layout: sidebar + content -->
      <div class="flex gap-6">
        <!-- Sidebar (desktop) -->
        <aside class="hidden w-56 shrink-0 lg:block">
          <nav aria-label="Server sections" class="space-y-0.5">
            <button
              v-for="section in sections"
              :key="section.key"
              :class="[
                'flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-left text-sm font-medium transition-colors',
                activeSection === section.key
                  ? 'bg-accent/10 text-accent'
                  : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground',
              ]"
              @click="navigateTo(section.key)"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-4 w-4 shrink-0"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="2"
              >
                <path stroke-linecap="round" stroke-linejoin="round" :d="section.icon" />
              </svg>
              {{ section.label }}
            </button>
          </nav>
        </aside>

        <!-- Content area -->
        <div class="min-w-0 flex-1">
          <Card class="rounded-xl border border-border/60 bg-card/50 backdrop-blur-sm py-0 shadow-none">
            <CardHeader class="border-b border-border/40 px-6 py-5">
              <div>
                <h2 class="text-lg font-semibold text-foreground">
                  {{ currentSection.label }}
                </h2>
                <p class="mt-0.5 text-sm text-muted-foreground">
                  {{ currentSection.description }}
                </p>
              </div>
            </CardHeader>

            <CardContent class="px-6 py-5">
              <!-- Section content -->
              <div class="space-y-6">
              <!-- Overview -->
              <div v-if="activeSection === 'overview'" class="space-y-4">
                <div class="grid gap-4 sm:grid-cols-2">
                  <div class="rounded-lg border border-border/60 bg-card/40 px-4 py-3">
                    <p class="text-xs font-medium text-muted-foreground">Status</p>
                    <div class="mt-1 flex items-center gap-2">
                      <span
                        class="h-2 w-2 rounded-full"
                        :class="{
                          'bg-primary': server.status === 'ready',
                          'bg-destructive': server.status === 'failed',
                          'bg-accent animate-pulse': server.status === 'provisioning' || server.status === 'pending',
                          'bg-muted-foreground': !['ready', 'failed', 'provisioning', 'pending'].includes(server.status),
                        }"
                      />
                      <span class="text-sm font-medium text-foreground/80 capitalize">{{ server.status }}</span>
                    </div>
                  </div>
                  <div class="rounded-lg border border-border/60 bg-card/40 px-4 py-3">
                    <p class="text-xs font-medium text-muted-foreground">Provider</p>
                    <p class="mt-1 text-sm font-medium text-foreground/80">{{ server.provider_type }}</p>
                  </div>
                  <div class="rounded-lg border border-border/60 bg-card/40 px-4 py-3">
                    <p class="text-xs font-medium text-muted-foreground">Location</p>
                    <p class="mt-1 text-sm font-medium text-foreground/80">{{ server.location }}</p>
                  </div>
                  <div class="rounded-lg border border-border/60 bg-card/40 px-4 py-3">
                    <p class="text-xs font-medium text-muted-foreground">Server Type</p>
                    <p class="mt-1 text-sm font-medium text-foreground/80">{{ server.server_type }}</p>
                  </div>
                  <div class="rounded-lg border border-border/60 bg-card/40 px-4 py-3">
                    <p class="text-xs font-medium text-muted-foreground">Profile</p>
                    <p class="mt-1 text-sm font-medium text-foreground/80">{{ server.profile_key }}</p>
                  </div>
                  <div class="rounded-lg border border-border/60 bg-card/40 px-4 py-3">
                    <p class="text-xs font-medium text-muted-foreground">Created</p>
                    <p class="mt-1 text-sm font-medium text-foreground/80">{{ formatDate(server.created_at) }}</p>
                  </div>
                </div>

                <!-- Provider Server ID (if available) -->
                <div v-if="server.provider_server_id" class="rounded-lg border border-border/60 bg-card/40 px-4 py-3">
                  <p class="text-xs font-medium text-muted-foreground">Provider Server ID</p>
                  <p class="mt-1 font-mono text-sm text-foreground/80">{{ server.provider_server_id }}</p>
                </div>

                <!-- Quick actions placeholder -->
                <div class="rounded-lg border border-dashed border-border/50 px-4 py-6 text-center">
                  <p class="text-sm text-muted-foreground">
                    Quick actions (reboot, stop, start, SSH access) will be available here.
                  </p>
                </div>
              </div>

              <!-- Sites -->
              <div v-if="activeSection === 'sites'" class="space-y-4">
                <div class="rounded-lg border border-dashed border-border/50 px-4 py-8 text-center">
                  <h3 class="text-sm font-medium text-foreground">No sites yet</h3>
                  <p class="mt-1 text-sm text-muted-foreground">
                    WordPress sites deployed to this server will appear here.
                  </p>
                  <p class="mt-3 text-xs text-muted-foreground">
                    Site management features are coming soon.
                  </p>
                </div>
              </div>

              <!-- Settings -->
              <div v-if="activeSection === 'settings'" class="space-y-4">
                <div class="grid gap-4 lg:grid-cols-2">
                  <div class="rounded-lg border border-border/60 bg-card/40 px-4 py-4">
                    <div>
                      <h3 class="text-sm font-semibold text-foreground">Rebuild server</h3>
                      <p class="mt-1 text-xs text-muted-foreground">
                        Reinstall the server with a new image.
                      </p>
                    </div>
                    <form class="mt-4 space-y-3" @submit.prevent="submitRebuild">
                      <div class="grid gap-3 sm:grid-cols-2">
                        <div class="space-y-1.5">
                          <Label class="text-xs font-medium text-muted-foreground">Server name</Label>
                          <Input v-model="rebuildServerName" placeholder="server-1" />
                        </div>
                        <div class="space-y-1.5">
                          <Label class="text-xs font-medium text-muted-foreground">Server image</Label>
                          <Input v-model="rebuildServerImage" placeholder="ubuntu-24.04" />
                        </div>
                      </div>
                      <div class="flex flex-wrap items-center gap-3">
                        <Button type="submit" size="sm" :disabled="rebuildState.loading">
                          Create rebuild job
                        </Button>
                        <span v-if="rebuildState.loading" class="text-xs text-muted-foreground">Submitting...</span>
                        <span v-if="rebuildState.success" class="text-xs text-primary">{{ rebuildState.success }}</span>
                        <span v-if="rebuildState.error" class="text-xs text-destructive">{{ rebuildState.error }}</span>
                      </div>
                    </form>
                  </div>

                  <div class="rounded-lg border border-border/60 bg-card/40 px-4 py-4">
                    <div>
                      <h3 class="text-sm font-semibold text-foreground">Resize server</h3>
                      <p class="mt-1 text-xs text-muted-foreground">
                        Update the server type and disk upgrade preference.
                      </p>
                    </div>
                    <form class="mt-4 space-y-3" @submit.prevent="submitResize">
                      <div class="grid gap-3 sm:grid-cols-2">
                        <div class="space-y-1.5">
                          <Label class="text-xs font-medium text-muted-foreground">Server type</Label>
                          <Input v-model="resizeServerType" placeholder="cpx31" />
                        </div>
                        <div class="space-y-1.5">
                          <Label class="text-xs font-medium text-muted-foreground">Upgrade disk</Label>
                          <div class="flex items-center gap-2">
                            <Switch v-model:checked="resizeUpgradeDisk" />
                            <span class="text-xs text-muted-foreground">
                              {{ resizeUpgradeDisk ? 'Yes' : 'No' }}
                            </span>
                          </div>
                        </div>
                      </div>
                      <div class="flex flex-wrap items-center gap-3">
                        <Button type="submit" size="sm" :disabled="resizeState.loading">
                          Create resize job
                        </Button>
                        <span v-if="resizeState.loading" class="text-xs text-muted-foreground">Submitting...</span>
                        <span v-if="resizeState.success" class="text-xs text-primary">{{ resizeState.success }}</span>
                        <span v-if="resizeState.error" class="text-xs text-destructive">{{ resizeState.error }}</span>
                      </div>
                    </form>
                  </div>

                  <div class="rounded-lg border border-border/60 bg-card/40 px-4 py-4">
                    <div>
                      <h3 class="text-sm font-semibold text-foreground">Update firewalls</h3>
                      <p class="mt-1 text-xs text-muted-foreground">
                        Replace firewall assignments using firewall names or IDs.
                      </p>
                    </div>
                    <form class="mt-4 space-y-3" @submit.prevent="submitUpdateFirewalls">
                      <div class="space-y-1.5">
                        <Label class="text-xs font-medium text-muted-foreground">Firewalls</Label>
                        <Input v-model="updateFirewallsList" placeholder="fw-core, fw-web" />
                        <p class="text-xs text-muted-foreground">Comma-separated list.</p>
                      </div>
                      <div class="flex flex-wrap items-center gap-3">
                        <Button type="submit" size="sm" :disabled="firewallsState.loading">
                          Create firewall update job
                        </Button>
                        <span v-if="firewallsState.loading" class="text-xs text-muted-foreground">Submitting...</span>
                        <span v-if="firewallsState.success" class="text-xs text-primary">{{ firewallsState.success }}</span>
                        <span v-if="firewallsState.error" class="text-xs text-destructive">{{ firewallsState.error }}</span>
                      </div>
                    </form>
                  </div>

                  <div class="rounded-lg border border-border/60 bg-card/40 px-4 py-4">
                    <div>
                      <h3 class="text-sm font-semibold text-foreground">Manage volume</h3>
                      <p class="mt-1 text-xs text-muted-foreground">
                        Attach or detach a volume for this server.
                      </p>
                    </div>
                    <form class="mt-4 space-y-3" @submit.prevent="submitManageVolume">
                      <div class="grid gap-3 sm:grid-cols-2">
                        <div class="space-y-1.5">
                          <Label class="text-xs font-medium text-muted-foreground">Volume name</Label>
                          <Input v-model="volumeName" placeholder="data-volume" />
                        </div>
                        <div class="space-y-1.5">
                          <Label class="text-xs font-medium text-muted-foreground">State</Label>
                          <Input v-model="volumeState" placeholder="present" />
                        </div>
                        <div v-if="volumeState === 'present'" class="space-y-1.5">
                          <Label class="text-xs font-medium text-muted-foreground">Size (GB)</Label>
                          <Input v-model="volumeSizeGb" placeholder="50" />
                        </div>
                        <div v-if="volumeState === 'present'" class="space-y-1.5">
                          <Label class="text-xs font-medium text-muted-foreground">Location</Label>
                          <Input v-model="volumeLocation" placeholder="fsn1" />
                        </div>
                        <div v-if="volumeState === 'present'" class="space-y-1.5">
                          <Label class="text-xs font-medium text-muted-foreground">Automount</Label>
                          <div class="flex items-center gap-2">
                            <Switch v-model:checked="volumeAutomount" />
                            <span class="text-xs text-muted-foreground">
                              {{ volumeAutomount ? 'Yes' : 'No' }}
                            </span>
                          </div>
                        </div>
                      </div>
                      <div class="flex flex-wrap items-center gap-3">
                        <Button type="submit" size="sm" :disabled="volumeStateUi.loading">
                          Create volume job
                        </Button>
                        <span v-if="volumeStateUi.loading" class="text-xs text-muted-foreground">Submitting...</span>
                        <span v-if="volumeStateUi.success" class="text-xs text-primary">{{ volumeStateUi.success }}</span>
                        <span v-if="volumeStateUi.error" class="text-xs text-destructive">{{ volumeStateUi.error }}</span>
                      </div>
                    </form>
                  </div>
                </div>
              </div>

              <!-- Activity -->
              <div v-if="activeSection === 'activity'" class="space-y-4">
                <ServerActivity :server-id="server.id" />
              </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </template>
  </div>
</template>
