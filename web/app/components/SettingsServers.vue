<script setup lang="ts">
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogFooter,
  DialogHeader,
  DialogScrollContent,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectItemText,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Spinner } from "@/components/ui/spinner"
import { cn } from "@/lib/utils"
import { useProviders } from "~/composables/useProviders"
import { useServers, type ServerTypePrice } from "~/composables/useServers"
import { useAllAgentStatus } from "~/composables/useAgentStatus"
import type { Job } from "~/composables/useJobs"
import { parseHealthResponse } from "~/lib/api-runtime"
import {
  inProgressServerStatuses,
  mutationBlockedServerStatuses,
  type CallbackURLMode,
  type ServerStatus,
  type SupportLevel,
  type SetupState,
} from "~/lib/platform-contract.generated"

const modal = useModal()

const { providers, fetchProviders } = useProviders()
const {
  servers,
  profiles,
  catalog,
  loading,
  saving,
  fetchServers,
  fetchCatalog,
  createServer,
  deleteServer,
} = useServers()

const { getStatusType, isConnected } = useAllAgentStatus({ pollInterval: 15000 })
const callbackMode = ref<CallbackURLMode>("unknown")
const callbackWarning = ref("")

// Delete confirmation state
const deleteConfirmId = ref<number | null>(null)
const deleting = ref(false)
const deleteError = ref("")
const deleteSuccess = ref("")

const formStep = ref<"configure" | "review" | "provisioning">("configure")
const formError = ref("")
const formLoadingCatalog = ref(false)

const formProviderId = ref("")
const formName = ref("")
const formLocation = ref("")
const formServerType = ref("")
const formProfileKey = ref("")

// Job tracking for provisioning step
const activeJobId = ref<number | null>(null)

const providerOptions = computed(() =>
  providers.value.map((p) => ({
    value: String(p.id),
    label: `${p.name} (${p.type})`,
  })),
)

const locationOptions = computed(() =>
  (catalog.value?.locations || []).map((loc) => ({
    value: loc.name,
    label: `${loc.name} - ${loc.description}`,
  })),
)

const selectableProfiles = computed(() =>
  profiles.value.filter((profile) => profile.support_level !== "unavailable"),
)

// Filter server types to only show those available at the selected location
const serverTypeOptions = computed(() => {
  const selectedLocation = formLocation.value
  return (catalog.value?.server_types || [])
    .filter((type_) => {
      // If no location selected, show all
      if (!selectedLocation) return true
      // Only show server types available at the selected location
      return type_.available_at?.includes(selectedLocation) ?? false
    })
    .map((type_) => {
      const priceLabel = formatMonthlyPrice(type_, selectedLocation)
      const detail = `${type_.cores} vCPU · ${type_.memory_gb}GB RAM · ${type_.disk_gb}GB SSD`
      return {
        value: type_.name,
        label: priceLabel
          ? `${type_.name} (${detail}, ${priceLabel}/mo)`
          : `${type_.name} (${detail})`,
      }
    })
})

const selectedProfile = computed(() =>
  profiles.value.find((profile) => profile.key === formProfileKey.value),
)

const defaultProfileKey = computed(() =>
  selectableProfiles.value[0]?.key || "",
)

const selectedTypeLabel = computed(() =>
  serverTypeOptions.value.find((option) => option.value === formServerType.value)?.label
  || formServerType.value,
)

const selectedProfileStatusClass = computed(() =>
  selectedProfile.value ? profileSupportClass(selectedProfile.value.support_level) : "border-border/60 bg-muted/40 text-muted-foreground"
)

const selectedProfileSupportText = computed(() => {
  if (!selectedProfile.value) {
    return ""
  }
  return selectedProfile.value.support_reason || supportLevelLabel(selectedProfile.value.support_level)
})

const controlClass =
  "w-full rounded-lg border border-border/60 bg-background/60 px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background"

const selectTriggerClass = cn(
  controlClass,
  "hover:border-border data-[placeholder]:text-muted-foreground",
)

const inputClass = cn(
  controlClass,
  "hover:border-border",
)

const selectContentClass =
  "border-border/60 bg-popover text-popover-foreground"

const selectItemClass =
  "text-foreground data-[disabled]:text-muted-foreground data-[highlighted]:bg-muted/60 data-[highlighted]:text-foreground"

const buttonBaseClass =
  "rounded-lg focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background"

const { apiFetch } = useApiClient()

onMounted(async () => {
  await Promise.all([fetchProviders(), fetchServers()])
  apiFetch('/health')
    .then((payload) => parseHealthResponse(payload))
    .then((body) => {
      callbackMode.value = body.callback_url_mode || "unknown"
      callbackWarning.value = body.callback_url_warning || ""
    })
    .catch(() => {})
})

const resetForm = () => {
  formStep.value = "configure"
  formError.value = ""
  formLoadingCatalog.value = false
  formProviderId.value = providerOptions.value[0]?.value || ""
  formName.value = ""
  formLocation.value = ""
  formServerType.value = ""
  formProfileKey.value = defaultProfileKey.value
  activeJobId.value = null
}

const openModal = async () => {
  resetForm()
  modal.open()
  if (formProviderId.value) {
    await loadCatalogForSelectedProvider()
  }
}

const loadCatalogForSelectedProvider = async () => {
  if (!formProviderId.value) {
    return
  }
  formLoadingCatalog.value = true
  formError.value = ""
  try {
    await fetchCatalog(Number(formProviderId.value))
    formLocation.value = locationOptions.value[0]?.value || ""
    // Server type will be set after location is selected (filtered by availability)
    formServerType.value = ""
    formProfileKey.value = defaultProfileKey.value
  } catch (e: any) {
    formError.value = e.message
  } finally {
    formLoadingCatalog.value = false
  }
}

watch(formProviderId, async () => {
  if (!modal.isOpen.value) {
    return
  }
  await loadCatalogForSelectedProvider()
})

// When location changes, reset server type to first available option
watch(formLocation, () => {
  // Set to first available server type for this location
  formServerType.value = serverTypeOptions.value[0]?.value || ""
})

const goToReview = () => {
  if (!isFormValid()) {
    formError.value = "Please fill all required fields before continuing."
    return
  }
  formError.value = ""
  formStep.value = "review"
}

const goBack = () => {
  formStep.value = "configure"
}

const submit = async () => {
  if (!isFormValid()) {
    formError.value = "Please complete the form before creating a server."
    return
  }

  formError.value = ""
  try {
    const result = await createServer({
      provider_id: Number(formProviderId.value),
      name: formName.value.trim(),
      location: formLocation.value,
      server_type: formServerType.value,
      profile_key: formProfileKey.value,
    })

    // Show provisioning progress instead of closing modal
    activeJobId.value = result.job_id
    formStep.value = "provisioning"

    // Refresh server list in background
    fetchServers()
  } catch (e: any) {
    formError.value = e.message
  }
}

const handleJobCompleted = (job: Job) => {
  // Refresh servers to show updated status
  fetchServers()
}

const handleJobFailed = (job: Job, error: string) => {
  // Refresh servers to show failed status
  fetchServers()
}

const closeAndReset = () => {
  modal.close()
  // Small delay to let modal animation complete before resetting
  setTimeout(resetForm, 200)
}

const isFormValid = () => {
  return (
    !!formProviderId.value
    && !!formName.value.trim()
    && !!formLocation.value
    && !!formServerType.value
    && !!formProfileKey.value
    && selectedProfile.value?.support_level !== "unavailable"
  )
}

const supportLevelLabel = (supportLevel: SupportLevel): string => {
  if (supportLevel === "supported") return "Supported"
  if (supportLevel === "experimental") return "Experimental"
  return "Not Ready"
}

const profileSupportClass = (supportLevel: SupportLevel): string => {
  if (supportLevel === "supported") return "border-primary/30 bg-primary/10 text-primary"
  if (supportLevel === "experimental") return "border-accent/30 bg-accent/10 text-accent"
  return "border-border/60 bg-muted/60 text-muted-foreground"
}

const formatMonthlyPrice = (
  serverType: { prices: ReadonlyArray<ServerTypePrice> },
  location: string,
): string => {
  const price = serverType.prices.find((entry) => entry.location_name === location)
  if (!price) return ""

  const amount = Number(price.monthly_gross)
  if (Number.isNaN(amount)) {
    return `${price.monthly_gross} ${price.currency}`
  }

  const formattedAmount = new Intl.NumberFormat("en-US", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount)
  return `${formattedAmount} ${price.currency}`
}

const formatDate = (iso: string): string => {
  try {
    return new Date(iso).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    })
  } catch {
    return iso
  }
}

const statusBadgeClass = (status: ServerStatus): string => {
  if (status === "ready") return "border-primary/30 bg-primary/10 text-primary"
  if (status === "failed") return "border-destructive/30 bg-destructive/10 text-destructive"
  if (inProgressServerStatuses.includes(status)) {
    return "border-accent/30 bg-accent/10 text-accent"
  }
  if (status === "deleted") return "border-border/60 bg-muted/60 text-muted-foreground"
  return "border-border/60 bg-muted/60 text-foreground"
}

const setupBadgeClass = (setupState?: SetupState): string => {
  if (setupState === "ready") return "border-primary/30 bg-primary/10 text-primary"
  if (setupState === "degraded") return "border-destructive/30 bg-destructive/10 text-destructive"
  if (setupState === "running") return "border-accent/30 bg-accent/10 text-accent"
  return "border-border/60 bg-muted/60 text-muted-foreground"
}

const confirmDelete = (serverId: number) => {
	deleteError.value = ""
	deleteSuccess.value = ""
  deleteConfirmId.value = serverId
}

const cancelDelete = () => {
  deleteConfirmId.value = null
  deleteError.value = ""
}

const executeDelete = async (serverId: number) => {
  deleting.value = true
  deleteError.value = ""
  deleteSuccess.value = ""
  try {
    const result = await deleteServer(serverId)
    deleteSuccess.value = `Delete job #${result.job_id} queued`
    deleteConfirmId.value = null
    await fetchServers()
  } catch (e: any) {
    deleteError.value = e.message || "Failed to queue delete job"
  } finally {
    deleting.value = false
  }
}

const handleDialogUpdate = (value: boolean) => {
  if (value) {
    modal.open()
    return
  }
  modal.close()
}
</script>

<template>
  <div class="space-y-6">
    <Alert
      v-if="callbackMode === 'ephemeral' && callbackWarning"
      class="border-accent/30 bg-accent/10 text-accent"
    >
      <AlertDescription>
        {{ callbackWarning }}
      </AlertDescription>
    </Alert>

    <div class="flex items-center justify-between">
      <p class="text-sm text-muted-foreground">
        Provision managed servers for agency WordPress workloads.
      </p>
      <Button
        size="sm"
        :class="cn(buttonBaseClass, 'bg-primary text-primary-foreground hover:bg-primary/90')"
        @click="openModal"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
        </svg>
        Create New Server
      </Button>
    </div>

    <div v-if="loading" class="flex items-center justify-center py-10">
      <Spinner class="text-muted-foreground" />
    </div>

    <div v-else-if="servers.length === 0" class="rounded-lg border border-dashed border-border/50 px-4 py-10 text-center">
      <h3 class="text-sm font-medium text-foreground">No servers yet</h3>
      <p class="mt-1 text-sm text-muted-foreground">
        Create your first managed server to start onboarding WordPress sites.
      </p>
    </div>

    <div v-else class="space-y-3">
        <div
          v-for="server in servers"
          :key="server.id"
        class="group flex items-center justify-between rounded-lg border border-border/60 bg-card/30 px-4 py-3 transition-colors hover:border-border/80 hover:bg-card/50"
      >
        <NuxtLink
          :to="`/servers/${server.id}`"
          class="min-w-0 flex-1"
        >
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium text-foreground group-hover:text-foreground">{{ server.name }}</span>
            <Badge :class="statusBadgeClass(server.status)">{{ server.status }}</Badge>
            <Badge :class="setupBadgeClass(server.setup_state)">setup {{ server.setup_state }}</Badge>
            <!-- Agent status indicator -->
            <span
              v-if="server.setup_state === 'ready'"
              class="flex items-center gap-1 text-xs"
              :title="'Agent: ' + getStatusType(server.id)"
            >
              <span
                class="h-1.5 w-1.5 rounded-full"
                :class="{
                  'bg-primary animate-pulse': getStatusType(server.id) === 'online',
                  'bg-amber-500': getStatusType(server.id) === 'unhealthy',
                  'bg-muted-foreground/50': getStatusType(server.id) === 'offline',
                  'bg-muted-foreground/30': getStatusType(server.id) === 'unknown',
                }"
              />
              <span
                class="hidden sm:inline"
                :class="{
                  'text-primary': getStatusType(server.id) === 'online',
                  'text-amber-500': getStatusType(server.id) === 'unhealthy',
                  'text-muted-foreground/60': getStatusType(server.id) === 'offline' || getStatusType(server.id) === 'unknown',
                }"
              >
                {{ isConnected(server.id) ? 'Agent' : '' }}
              </span>
            </span>
          </div>
          <p class="text-xs text-muted-foreground">
            {{ server.location }} · {{ server.server_type }} · {{ server.profile_key }} · Added {{ formatDate(server.created_at) }}
          </p>
          <p v-if="server.setup_state === 'degraded' && server.setup_last_error" class="mt-1 text-xs text-destructive">
            Setup needs attention: {{ server.setup_last_error }}
          </p>
          <p v-if="server.status === 'deleting'" class="mt-1 text-xs text-accent">
            Deletion is queued and runs asynchronously until provider-side removal completes.
          </p>
          <p v-else-if="server.status === 'deleted'" class="mt-1 text-xs text-muted-foreground">
            Tombstone record retained after provider-side deletion.
          </p>
        </NuxtLink>
        <div class="flex items-center gap-3">
          <span class="text-xs text-muted-foreground">{{ server.provider_type }}</span>
          <!-- Delete button (always visible for failed, hover for others) -->
          <Button
            v-if="deleteConfirmId !== server.id"
            variant="ghost"
            size="icon-sm"
            type="button"
            :class="cn(
              buttonBaseClass,
              'text-muted-foreground hover:bg-destructive/10 hover:text-destructive',
              !['failed', 'ready'].includes(server.status) && 'opacity-0 group-hover:opacity-100',
            )"
            :disabled="mutationBlockedServerStatuses.includes(server.status)"
            title="Delete server"
            @click.prevent="confirmDelete(server.id)"
          >
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </Button>
          <!-- Delete confirmation -->
          <div v-else class="flex items-center gap-1">
            <Button
              variant="ghost"
              type="button"
              :disabled="deleting"
              :class="cn(buttonBaseClass, 'h-7 px-2 text-xs text-destructive hover:bg-destructive/10')"
              @click.prevent="executeDelete(server.id)"
            >
              {{ deleting ? "Deleting..." : "Confirm" }}
            </Button>
            <Button
              variant="ghost"
              type="button"
              :disabled="deleting"
              :class="cn(buttonBaseClass, 'h-7 px-2 text-xs text-muted-foreground hover:bg-muted/50 hover:text-foreground')"
              @click.prevent="cancelDelete"
            >
              Cancel
            </Button>
          </div>
        </div>
      </div>
      <p v-if="deleteSuccess" class="text-xs text-primary">{{ deleteSuccess }}</p>
      <p v-if="deleteError" class="text-xs text-destructive">{{ deleteError }}</p>
    </div>

    <Dialog
      v-if="modal.isOpen.value"
      :open="modal.isOpen.value"
      @update:open="handleDialogUpdate"
    >
      <DialogScrollContent
        class="max-h-[85vh] border-border/60 bg-popover/90 p-5 text-popover-foreground sm:max-w-xl sm:p-6"
      >
        <DialogHeader class="text-left">
          <DialogTitle class="text-base font-semibold text-foreground">
            Create Managed Server
          </DialogTitle>
        </DialogHeader>

        <div class="space-y-4">
          <Alert
            v-if="formError"
            variant="destructive"
            class="border-destructive/30 bg-destructive/10 text-destructive"
          >
            <AlertDescription class="text-destructive">
              {{ formError }}
            </AlertDescription>
          </Alert>

          <template v-if="formStep === 'configure'">
            <div class="space-y-1.5">
              <Label class="text-sm font-medium text-muted-foreground">
                Provider Connection
              </Label>
              <Select v-model="formProviderId">
                <SelectTrigger :class="selectTriggerClass">
                  <SelectValue placeholder="Select provider" />
                </SelectTrigger>
                <SelectContent :class="selectContentClass">
                  <SelectItem
                    v-for="option in providerOptions"
                    :key="option.value"
                    :value="option.value"
                    :class="selectItemClass"
                  >
                    <SelectItemText>{{ option.label }}</SelectItemText>
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div class="space-y-1.5">
              <Label class="text-sm font-medium text-muted-foreground">
                Server Name
              </Label>
              <Input
                v-model="formName"
                placeholder="e.g. agency-prod-eu-1"
                :class="inputClass"
              />
            </div>

            <div class="space-y-1.5">
              <Label class="text-sm font-medium text-muted-foreground">
                Server Profile
              </Label>
              <Select v-model="formProfileKey" :disabled="formLoadingCatalog">
                <SelectTrigger :class="selectTriggerClass">
                  <SelectValue placeholder="Select server profile">
                    <div v-if="selectedProfile" class="flex min-w-0 items-center justify-between gap-2">
                      <span class="truncate text-sm text-foreground">{{ selectedProfile.name }}</span>
                      <Badge :class="profileSupportClass(selectedProfile.support_level)">
                        {{ supportLevelLabel(selectedProfile.support_level) }}
                      </Badge>
                    </div>
                  </SelectValue>
                </SelectTrigger>
                <SelectContent :class="selectContentClass">
                  <SelectItem
                    v-for="profile in profiles"
                    :key="profile.key"
                    :value="profile.key"
                    :disabled="profile.support_level === 'unavailable'"
                    :class="selectItemClass"
                  >
                    <div class="flex w-full min-w-0 items-center justify-between gap-2">
                      <span class="truncate">{{ profile.name }}</span>
                      <Badge :class="profileSupportClass(profile.support_level)">
                        {{ supportLevelLabel(profile.support_level) }}
                      </Badge>
                    </div>
                  </SelectItem>
                </SelectContent>
              </Select>
              <div
                v-if="selectedProfile"
                class="rounded-lg border px-3 py-3"
                :class="selectedProfileStatusClass"
              >
                <ul class="space-y-2 text-sm text-muted-foreground">
                  <li class="flex items-start gap-2">
                    <span class="mt-[0.35rem] h-1.5 w-1.5 shrink-0 rounded-full bg-current opacity-70" />
                    <span><strong class="text-foreground">Description:</strong> {{ selectedProfile.description }}</span>
                  </li>
                  <li class="flex items-start gap-2">
                    <span class="mt-[0.35rem] h-1.5 w-1.5 shrink-0 rounded-full bg-current opacity-70" />
                    <span><strong class="text-foreground">Configure guarantee:</strong> {{ selectedProfile.configure_guarantee }}</span>
                  </li>
                  <li class="flex items-start gap-2">
                    <span class="mt-[0.35rem] h-1.5 w-1.5 shrink-0 rounded-full bg-current opacity-70" />
                    <span><strong class="text-foreground">Support:</strong> {{ selectedProfileSupportText }}</span>
                  </li>
                </ul>
              </div>
              <p v-else class="text-xs text-muted-foreground">
                No selectable server profile is available for this provider yet.
              </p>
            </div>

            <div class="space-y-1.5">
              <Label class="text-sm font-medium text-muted-foreground">
                Region
              </Label>
              <Select v-model="formLocation" :disabled="formLoadingCatalog">
                <SelectTrigger :class="selectTriggerClass">
                  <SelectValue placeholder="Select region" />
                </SelectTrigger>
                <SelectContent :class="selectContentClass">
                  <SelectItem
                    v-for="option in locationOptions"
                    :key="option.value"
                    :value="option.value"
                    :class="selectItemClass"
                  >
                    <SelectItemText>{{ option.label }}</SelectItemText>
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div class="space-y-1.5">
              <Label class="text-sm font-medium text-muted-foreground">
                Size
              </Label>
              <Select
                v-model="formServerType"
                :disabled="formLoadingCatalog || !formLocation"
              >
                <SelectTrigger :class="selectTriggerClass">
                  <SelectValue placeholder="Select size" />
                </SelectTrigger>
                <SelectContent :class="selectContentClass">
                  <SelectItem
                    v-for="option in serverTypeOptions"
                    :key="option.value"
                    :value="option.value"
                    :class="selectItemClass"
                  >
                    <SelectItemText>{{ option.label }}</SelectItemText>
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <p v-if="formLoadingCatalog" class="text-xs text-muted-foreground">
              Loading provider catalog...
            </p>
          </template>

          <template v-else-if="formStep === 'review'">
            <div class="rounded-lg border border-border/60 bg-muted/40 px-4 py-3 text-sm text-muted-foreground">
              <p><strong class="text-foreground">Name:</strong> {{ formName }}</p>
              <p><strong class="text-foreground">Region:</strong> {{ formLocation }}</p>
              <p><strong class="text-foreground">Size:</strong> {{ selectedTypeLabel }}</p>
              <div class="mt-1">
                <p><strong class="text-foreground">Profile:</strong> {{ selectedProfile?.name || formProfileKey }}</p>
                <div v-if="selectedProfile" class="mt-2 flex flex-wrap items-center gap-2">
                  <Badge :class="profileSupportClass(selectedProfile.support_level)">
                    {{ supportLevelLabel(selectedProfile.support_level) }}
                  </Badge>
                </div>
                <p v-if="selectedProfile" class="mt-2 text-xs text-muted-foreground">
                  {{ selectedProfileSupportText }}
                </p>
              </div>
            </div>
            <p class="text-xs text-muted-foreground">
              The base image is determined by the selected profile. Advanced networking, firewalls, and storage options are intentionally hidden for this managed flow.
            </p>
          </template>

          <!-- Provisioning step: show job timeline -->
          <template v-else-if="formStep === 'provisioning' && activeJobId">
            <div class="space-y-3">
              <div class="flex items-center gap-2">
                <div class="h-2 w-2 animate-pulse rounded-full bg-accent" />
                <span class="text-sm font-medium text-foreground/80">Provisioning {{ formName }}</span>
              </div>
              <JobTimeline
                :job-id="activeJobId"
                compact
                @completed="handleJobCompleted"
                @failed="handleJobFailed"
              />
            </div>
          </template>
        </div>

        <DialogFooter class="flex-row items-center justify-end gap-2">
          <!-- Cancel/Close button -->
          <Button
            v-if="formStep !== 'provisioning'"
            variant="ghost"
            size="sm"
            :class="cn(buttonBaseClass, 'text-muted-foreground hover:bg-muted/50')"
            @click="modal.close"
          >
            Cancel
          </Button>

          <!-- Back button (review step only) -->
          <Button
            v-if="formStep === 'review'"
            variant="ghost"
            size="sm"
            :class="cn(buttonBaseClass, 'text-muted-foreground hover:bg-muted/50')"
            @click="goBack"
          >
            Back
          </Button>

          <!-- Review button (configure step) -->
          <Button
            v-if="formStep === 'configure'"
            size="sm"
            :disabled="!isFormValid() || formLoadingCatalog"
            :class="cn(buttonBaseClass, 'bg-primary text-primary-foreground hover:bg-primary/90')"
            @click="goToReview"
          >
            Review
          </Button>

          <!-- Create button (review step) -->
          <Button
            v-if="formStep === 'review'"
            size="sm"
            :disabled="saving"
            :class="cn(buttonBaseClass, 'bg-primary text-primary-foreground hover:bg-primary/90')"
            @click="submit"
          >
            <Spinner v-if="saving" class="text-primary-foreground" />
            Create Server
          </Button>

          <!-- Done button (provisioning step) -->
          <Button
            v-if="formStep === 'provisioning'"
            size="sm"
            :class="cn(buttonBaseClass, 'bg-primary text-primary-foreground hover:bg-primary/90')"
            @click="closeAndReset"
          >
            Done
          </Button>
        </DialogFooter>
      </DialogScrollContent>
    </Dialog>
  </div>
</template>
