<script setup lang="ts">
import { useProviders } from '~/composables/useProviders'
import { useServers, type ServerTypeOption, type ServerTypePrice } from '~/composables/useServers'

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
} = useServers()

const formStep = ref<'configure' | 'review'>('configure')
const formError = ref('')
const formLoadingCatalog = ref(false)

const formProviderId = ref('')
const formName = ref('')
const formLocation = ref('')
const formServerType = ref('')
const formImage = ref('')
const formProfileKey = ref('')

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

const imageOptions = computed(() =>
  (catalog.value?.images || []).map((img) => ({
    value: img.name,
    label: img.os_version ? `${img.name} (${img.os_version})` : img.name,
  })),
)

const profileOptions = computed(() =>
  profiles.value.map((profile) => ({
    value: profile.key,
    label: profile.name,
  })),
)

const serverTypeOptions = computed(() =>
  (catalog.value?.server_types || []).map((type_) => {
    const priceLabel = formatMonthlyPrice(type_, formLocation.value)
    const detail = `${type_.cores} vCPU · ${type_.memory_gb}GB RAM · ${type_.disk_gb}GB SSD`
    return {
      value: type_.name,
      label: priceLabel ? `${type_.name} (${detail}, ${priceLabel}/mo)` : `${type_.name} (${detail})`,
    }
  }),
)

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
  formImage.value = ''
  formProfileKey.value = ''
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
    formServerType.value = serverTypeOptions.value[0]?.value || ''
    formImage.value = imageOptions.value[0]?.value || ''
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
    await createServer({
      provider_id: Number(formProviderId.value),
      name: formName.value.trim(),
      location: formLocation.value,
      server_type: formServerType.value,
      image: formImage.value,
      profile_key: formProfileKey.value,
    })
    modal.close()
    await fetchServers()
  } catch (e: any) {
    formError.value = e.message
  }
}

const isFormValid = () => {
  return (
    !!formProviderId.value &&
    !!formName.value.trim() &&
    !!formLocation.value &&
    !!formServerType.value &&
    !!formImage.value &&
    !!formProfileKey.value
  )
}

const formatMonthlyPrice = (
  serverType: { prices: ReadonlyArray<ServerTypePrice> },
  location: string,
): string => {
  const price = serverType.prices.find((entry) => entry.location_name === location)
  if (!price) return ''
  return `${price.monthly_gross} ${price.currency}`
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
        class="flex items-center justify-between rounded-lg border border-surface-800/60 bg-surface-900/30 px-4 py-3"
      >
        <div>
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium text-surface-100">{{ server.name }}</span>
            <UiBadge :variant="statusVariant(server.status)">{{ server.status }}</UiBadge>
          </div>
          <p class="text-xs text-surface-500">
            {{ server.location }} · {{ server.server_type }} · {{ server.profile_key }} · Added {{ formatDate(server.created_at) }}
          </p>
        </div>
        <span class="text-xs text-surface-500">{{ server.provider_type }}</span>
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
            :disabled="formLoadingCatalog"
          />

          <UiSelect
            v-model="formImage"
            label="Base Image"
            :options="imageOptions"
            placeholder="Select image"
            :disabled="formLoadingCatalog"
          />

          <p v-if="formLoadingCatalog" class="text-xs text-surface-500">Loading provider catalog...</p>
        </template>

        <template v-else>
          <div class="rounded-lg border border-surface-800/60 bg-surface-950/40 px-4 py-3 text-sm text-surface-300">
            <p><strong class="text-surface-100">Name:</strong> {{ formName }}</p>
            <p><strong class="text-surface-100">Region:</strong> {{ formLocation }}</p>
            <p><strong class="text-surface-100">Size:</strong> {{ selectedTypeLabel }}</p>
            <p><strong class="text-surface-100">Image:</strong> {{ formImage }}</p>
            <p><strong class="text-surface-100">Profile:</strong> {{ selectedProfile?.name || formProfileKey }}</p>
          </div>
          <p class="text-xs text-surface-500">
            Advanced networking, firewalls, and storage options are intentionally hidden for this managed flow.
          </p>
        </template>
      </div>

      <template #footer>
        <div class="flex items-center justify-end gap-2">
          <UiButton variant="ghost" size="sm" @click="modal.close">Cancel</UiButton>

          <UiButton
            v-if="formStep === 'review'"
            variant="ghost"
            size="sm"
            @click="goBack"
          >
            Back
          </UiButton>

          <UiButton
            v-if="formStep === 'configure'"
            size="sm"
            :disabled="!isFormValid() || formLoadingCatalog"
            @click="goToReview"
          >
            Review
          </UiButton>

          <UiButton
            v-if="formStep === 'review'"
            size="sm"
            :loading="saving"
            @click="submit"
          >
            Create Server
          </UiButton>
        </div>
      </template>
    </UiModal>
  </div>
</template>
