<script setup lang="ts">
import { useProviders } from '~/composables/useProviders'
import { useServers, type ServerTypePrice } from '~/composables/useServers'
import type { Job } from '~/composables/useJobs'

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

// Delete confirmation state
const deleteConfirmId = ref<number | null>(null)
const deleting = ref(false)

const formStep = ref<'configure' | 'review' | 'provisioning'>('configure')
const formError = ref('')
const formLoadingCatalog = ref(false)

const formProviderId = ref('')
const formName = ref('')
const formLocation = ref('')
const formServerType = ref('')
const formProfileKey = ref('')

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

const profileOptions = computed(() =>
  profiles.value.map((profile) => ({
    value: profile.key,
    label: profile.name,
  })),
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
        label: priceLabel ? `${type_.name} (${detail}, ${priceLabel}/mo)` : `${type_.name} (${detail})`,
      }
    })
})

const selectedProfile = computed(() =>
  profiles.value.find((profile) => profile.key === formProfileKey.value),
)

const selectedTypeLabel = computed(() =>
  serverTypeOptions.value.find((option) => option.value === formServerType.value)?.label || formServerType.value,
)

onMounted(async () => {
  await Promise.all([fetchProviders(), fetchServers()])
})

const resetForm = () => {
  formStep.value = 'configure'
  formError.value = ''
  formLoadingCatalog.value = false
  formProviderId.value = providerOptions.value[0]?.value || ''
  formName.value = ''
  formLocation.value = ''
  formServerType.value = ''
  formProfileKey.value = ''
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
  formError.value = ''
  try {
    await fetchCatalog(Number(formProviderId.value))
    formLocation.value = locationOptions.value[0]?.value || ''
    // Server type will be set after location is selected (filtered by availability)
    formServerType.value = ''
    formProfileKey.value = profileOptions.value[0]?.value || ''
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
  formServerType.value = serverTypeOptions.value[0]?.value || ''
})

const goToReview = () => {
  if (!isFormValid()) {
    formError.value = 'Please fill all required fields before continuing.'
    return
  }
  formError.value = ''
  formStep.value = 'review'
}

const goBack = () => {
  formStep.value = 'configure'
}

const submit = async () => {
  if (!isFormValid()) {
    formError.value = 'Please complete the form before creating a server.'
    return
  }

  formError.value = ''
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
    formStep.value = 'provisioning'
    
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
    !!formProviderId.value &&
    !!formName.value.trim() &&
    !!formLocation.value &&
    !!formServerType.value &&
    !!formProfileKey.value
  )
}

const formatMonthlyPrice = (
  serverType: { prices: ReadonlyArray<ServerTypePrice> },
  location: string,
): string => {
  const price = serverType.prices.find((entry) => entry.location_name === location)
  if (!price) return ''

  const amount = Number(price.monthly_gross)
  if (Number.isNaN(amount)) {
    return `${price.monthly_gross} ${price.currency}`
  }

  const formattedAmount = new Intl.NumberFormat('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount)
  return `${formattedAmount} ${price.currency}`
}

const formatDate = (iso: string): string => {
  try {
    return new Date(iso).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    })
  } catch {
    return iso
  }
}

const statusVariant = (status: string): 'success' | 'warning' | 'danger' | 'default' => {
  if (status === 'ready') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'provisioning' || status === 'pending') return 'warning'
  return 'default'
}

const confirmDelete = (serverId: number) => {
  deleteConfirmId.value = serverId
}

const cancelDelete = () => {
  deleteConfirmId.value = null
}

const executeDelete = async (serverId: number) => {
  deleting.value = true
  try {
    await deleteServer(serverId)
    deleteConfirmId.value = null
    await fetchServers()
  } catch (e: any) {
    // Could show a toast here
    console.error('Failed to delete server:', e.message)
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <p class="text-sm text-surface-400">
        Provision managed servers for agency WordPress workloads.
      </p>
      <UiButton size="sm" @click="openModal">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
        </svg>
        Create New Server
      </UiButton>
    </div>

    <div v-if="loading" class="flex items-center justify-center py-10">
      <svg class="h-5 w-5 animate-spin text-surface-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
      </svg>
    </div>

    <div v-else-if="servers.length === 0" class="rounded-lg border border-dashed border-surface-700/50 px-4 py-10 text-center">
      <h3 class="text-sm font-medium text-surface-200">No servers yet</h3>
      <p class="mt-1 text-sm text-surface-500">
        Create your first managed server to start onboarding WordPress sites.
      </p>
    </div>

    <div v-else class="space-y-3">
      <div
        v-for="server in servers"
        :key="server.id"
        class="group flex items-center justify-between rounded-lg border border-surface-800/60 bg-surface-900/30 px-4 py-3 transition-colors hover:border-surface-700/60 hover:bg-surface-900/50"
      >
        <NuxtLink
          :to="`/servers/${server.id}`"
          class="min-w-0 flex-1"
        >
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium text-surface-100 group-hover:text-surface-50">{{ server.name }}</span>
            <UiBadge :variant="statusVariant(server.status)">{{ server.status }}</UiBadge>
          </div>
          <p class="text-xs text-surface-500">
            {{ server.location }} · {{ server.server_type }} · {{ server.profile_key }} · Added {{ formatDate(server.created_at) }}
          </p>
        </NuxtLink>
        <div class="flex items-center gap-3">
          <span class="text-xs text-surface-500">{{ server.provider_type }}</span>
          <!-- Delete button (always visible for failed, hover for others) -->
          <button
            v-if="deleteConfirmId !== server.id"
            class="rounded p-1 text-surface-500 transition-colors hover:bg-danger-900/30 hover:text-danger-400"
            :class="{ 'opacity-0 group-hover:opacity-100': server.status !== 'failed' }"
            title="Delete server"
            @click.prevent="confirmDelete(server.id)"
          >
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
          <!-- Delete confirmation -->
          <div v-else class="flex items-center gap-1">
            <button
              class="rounded px-2 py-1 text-xs font-medium text-danger-400 transition-colors hover:bg-danger-900/30"
              :disabled="deleting"
              @click.prevent="executeDelete(server.id)"
            >
              {{ deleting ? 'Deleting...' : 'Confirm' }}
            </button>
            <button
              class="rounded px-2 py-1 text-xs text-surface-400 transition-colors hover:bg-surface-800/50 hover:text-surface-200"
              :disabled="deleting"
              @click.prevent="cancelDelete"
            >
              Cancel
            </button>
          </div>
        </div>
      </div>
    </div>

    <UiModal :open="modal.isOpen.value" title="Create Managed Server" @close="modal.close">
      <div class="space-y-4">
        <div v-if="formError" class="rounded-lg border border-danger-600/30 bg-danger-900/20 px-3 py-2 text-sm text-danger-300">
          {{ formError }}
        </div>

        <template v-if="formStep === 'configure'">
          <UiSelect
            v-model="formProviderId"
            label="Provider Connection"
            :options="providerOptions"
            placeholder="Select provider"
          />

          <UiInput
            v-model="formName"
            label="Server Name"
            placeholder="e.g. agency-prod-eu-1"
          />

          <UiSelect
            v-model="formProfileKey"
            label="Server Profile"
            :options="profileOptions"
            placeholder="Select profile"
            :disabled="formLoadingCatalog"
          />

          <UiSelect
            v-model="formLocation"
            label="Region"
            :options="locationOptions"
            placeholder="Select region"
            :disabled="formLoadingCatalog"
          />

          <UiSelect
            v-model="formServerType"
            label="Size"
            :options="serverTypeOptions"
            placeholder="Select size"
            :disabled="formLoadingCatalog || !formLocation"
          />

          <p v-if="formLoadingCatalog" class="text-xs text-surface-500">Loading provider catalog...</p>
        </template>

        <template v-else-if="formStep === 'review'">
          <div class="rounded-lg border border-surface-800/60 bg-surface-950/40 px-4 py-3 text-sm text-surface-300">
            <p><strong class="text-surface-100">Name:</strong> {{ formName }}</p>
            <p><strong class="text-surface-100">Region:</strong> {{ formLocation }}</p>
            <p><strong class="text-surface-100">Size:</strong> {{ selectedTypeLabel }}</p>
            <p><strong class="text-surface-100">Profile:</strong> {{ selectedProfile?.name || formProfileKey }}</p>
          </div>
          <p class="text-xs text-surface-500">
            The base image is determined by the selected profile. Advanced networking, firewalls, and storage options are intentionally hidden for this managed flow.
          </p>
        </template>

        <!-- Provisioning step: show job timeline -->
        <template v-else-if="formStep === 'provisioning' && activeJobId">
          <div class="space-y-3">
            <div class="flex items-center gap-2">
              <div class="h-2 w-2 animate-pulse rounded-full bg-accent-500" />
              <span class="text-sm font-medium text-surface-200">Provisioning {{ formName }}</span>
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

      <template #footer>
        <div class="flex items-center justify-end gap-2">
          <!-- Cancel/Close button -->
          <UiButton
            v-if="formStep !== 'provisioning'"
            variant="ghost"
            size="sm"
            @click="modal.close"
          >
            Cancel
          </UiButton>

          <!-- Back button (review step only) -->
          <UiButton
            v-if="formStep === 'review'"
            variant="ghost"
            size="sm"
            @click="goBack"
          >
            Back
          </UiButton>

          <!-- Review button (configure step) -->
          <UiButton
            v-if="formStep === 'configure'"
            size="sm"
            :disabled="!isFormValid() || formLoadingCatalog"
            @click="goToReview"
          >
            Review
          </UiButton>

          <!-- Create button (review step) -->
          <UiButton
            v-if="formStep === 'review'"
            size="sm"
            :loading="saving"
            @click="submit"
          >
            Create Server
          </UiButton>

          <!-- Done button (provisioning step) -->
          <UiButton
            v-if="formStep === 'provisioning'"
            size="sm"
            @click="closeAndReset"
          >
            Done
          </UiButton>
        </div>
      </template>
    </UiModal>
  </div>
</template>
