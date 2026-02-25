<script setup lang="ts">
import { useProviders, type ValidationResult } from '~/composables/useProviders'
import { cn } from '@/lib/utils'
import { Alert } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'

const {
  providers,
  loading,
  fetchProviders,
  fetchProviderTypes,
  validateToken,
  createProvider,
  deleteProvider,
} = useProviders()

const modal = useModal()

// Form state
const formType = ref('hetzner')
const formName = ref('')
const formToken = ref('')
const formStep = ref<'configure' | 'validated'>('configure')
const formValidating = ref(false)
const formSaving = ref(false)
const formError = ref('')
const formValidation = ref<ValidationResult | null>(null)

// Delete confirmation
const deletingId = ref<number | null>(null)

onMounted(async () => {
  await Promise.all([fetchProviders(), fetchProviderTypes()])
})

const resetForm = () => {
  formType.value = 'hetzner'
  formName.value = ''
  formToken.value = ''
  formStep.value = 'configure'
  formValidating.value = false
  formSaving.value = false
  formError.value = ''
  formValidation.value = null
}

const openAddModal = () => {
  resetForm()
  modal.open()
}

const handleDialogChange = (value: boolean) => {
  if (value) {
    modal.open()
  } else {
    modal.close()
  }
}

const handleValidate = async () => {
  formError.value = ''
  formValidating.value = true
  formValidation.value = null

  try {
    const result = await validateToken(formType.value, formToken.value)
    formValidation.value = result

    if (result.valid && result.read_write) {
      formStep.value = 'validated'
    }
  } catch (e: any) {
    formError.value = e.message
  } finally {
    formValidating.value = false
  }
}

const handleSave = async () => {
  if (!formName.value.trim()) {
    formError.value = 'Please enter a name for this provider connection.'
    return
  }

  formError.value = ''
  formSaving.value = true

  try {
    await createProvider(formType.value, formName.value.trim(), formToken.value)
    modal.close()
    await fetchProviders()
  } catch (e: any) {
    formError.value = e.message
  } finally {
    formSaving.value = false
  }
}

const handleDelete = async (id: number) => {
  deletingId.value = id
  try {
    await deleteProvider(id)
    await fetchProviders()
  } catch (e: any) {
    formError.value = e.message
  } finally {
    deletingId.value = null
  }
}

const providerDisplayName = (type_: string): string => {
  const names: Record<string, string> = { hetzner: 'Hetzner Cloud' }
  return names[type_] || type_
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

const inputClass = (hasError: boolean) => cn(
  'w-full rounded-lg border bg-surface-900/60 px-3 py-2 text-sm text-surface-100 placeholder:text-surface-400 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-surface-950',
  hasError
    ? 'border-danger-500/60 focus-visible:ring-danger-500/40'
    : 'border-surface-700/60 focus-visible:ring-accent-500/60 hover:border-surface-600',
)

const statusBadgeClass = (status: string) => cn(
  'border-surface-700/60 bg-surface-800/60 text-surface-100',
  status === 'active' && 'border-success-700/40 bg-success-900/40 text-success-300',
)

const validationClass = (result: ValidationResult) => cn(
  'rounded-lg border px-3.5 py-2.5 text-sm',
  result.valid && result.read_write
    ? 'border-success-600/30 bg-success-900/20 text-success-400'
    : result.valid && !result.read_write
      ? 'border-warning-600/30 bg-warning-900/20 text-warning-400'
      : 'border-danger-600/30 bg-danger-900/20 text-danger-400',
)
</script>

<template>
  <div class="space-y-6">
    <!-- Header row -->
    <div class="flex items-center justify-between">
      <p class="text-sm text-surface-400">
        Connect cloud providers to provision and manage servers.
      </p>
      <Button
        size="sm"
        class="bg-accent-500 text-surface-950 hover:bg-accent-400"
        @click="openAddModal"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
        </svg>
        Add Provider
      </Button>
    </div>

    <!-- Loading state -->
    <div v-if="loading" class="flex items-center justify-center py-12">
      <Spinner class="size-5 text-surface-400" />
    </div>

    <!-- Empty state -->
    <div v-else-if="providers.length === 0" class="rounded-lg border border-dashed border-surface-700/50 px-4 py-12 text-center">
      <svg xmlns="http://www.w3.org/2000/svg" class="mx-auto h-10 w-10 text-surface-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
        <path stroke-linecap="round" stroke-linejoin="round" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
      </svg>
      <h3 class="mt-3 text-sm font-medium text-surface-200">No providers connected</h3>
      <p class="mt-1 text-sm text-surface-500">
        Add a cloud provider to start provisioning servers.
      </p>
      <div class="mt-4">
        <Button
          size="sm"
          variant="outline"
          class="border-surface-700/60 text-surface-100 hover:bg-surface-800/60"
          @click="openAddModal"
        >
          Add your first provider
        </Button>
      </div>
    </div>

    <!-- Provider list -->
    <div v-else class="space-y-3">
      <div
        v-for="p in providers"
        :key="p.id"
        class="flex items-center justify-between rounded-lg border border-surface-800/60 bg-surface-900/30 px-4 py-3 transition-colors hover:bg-surface-900/50"
      >
        <div class="flex items-center gap-3">
          <!-- Hetzner logo placeholder -->
          <div class="flex h-9 w-9 items-center justify-center rounded-lg bg-surface-800/80 text-xs font-bold text-surface-300">
            {{ p.type === 'hetzner' ? 'Hz' : p.type.slice(0, 2).toUpperCase() }}
          </div>
          <div>
            <div class="flex items-center gap-2">
              <span class="text-sm font-medium text-surface-100">{{ p.name }}</span>
              <Badge :class="statusBadgeClass(p.status)">
                {{ p.status }}
              </Badge>
            </div>
            <span class="text-xs text-surface-500">
              {{ providerDisplayName(p.type) }} &middot; Added {{ formatDate(p.created_at) }}
            </span>
          </div>
        </div>

        <Button
          size="icon-sm"
          variant="ghost"
          class="text-surface-400 hover:text-danger-400"
          :disabled="deletingId === p.id"
          @click="handleDelete(p.id)"
        >
          <Spinner v-if="deletingId === p.id" class="text-surface-400" />
          <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
          </svg>
        </Button>
      </div>
    </div>

    <!-- Add Provider Modal -->
    <Dialog :open="modal.isOpen.value" @update:open="handleDialogChange">
      <DialogContent
        v-if="modal.isOpen.value"
        :show-close-button="false"
        :class="cn('w-full max-w-lg rounded-xl border border-surface-800/60 bg-surface-900/80 p-0 shadow-2xl')"
      >
        <DialogHeader class="flex flex-row items-center justify-between border-b border-surface-800/40 px-6 py-4 text-left">
          <DialogTitle class="text-base font-semibold text-surface-100">Add Cloud Provider</DialogTitle>
          <Button
            type="button"
            size="icon-sm"
            variant="ghost"
            class="text-surface-300 hover:bg-surface-800/60 hover:text-surface-100"
            aria-label="Close modal"
            @click="modal.close"
          >
            <span aria-hidden="true">&times;</span>
          </Button>
        </DialogHeader>

        <div class="px-6 py-5">
          <div class="space-y-5">
            <!-- Step 1: Configure -->
            <template v-if="formStep === 'configure'">
              <!-- Provider type (only Hetzner for now) -->
              <div class="space-y-1.5">
                <Label class="text-sm font-medium text-surface-300">Provider</Label>
                <div class="flex items-center gap-3 rounded-lg border border-accent-500/30 bg-accent-500/5 px-3.5 py-2.5">
                  <div class="flex h-8 w-8 items-center justify-center rounded-md bg-surface-800 text-xs font-bold text-surface-200">
                    Hz
                  </div>
                  <div>
                    <span class="text-sm font-medium text-surface-100">Hetzner Cloud</span>
                    <p class="text-xs text-surface-400">European cloud infrastructure</p>
                  </div>
                </div>
              </div>

              <!-- Tutorial -->
              <div class="rounded-lg border border-surface-800/60 bg-surface-950/50 px-4 py-3">
                <h4 class="text-xs font-semibold uppercase tracking-wider text-surface-400">How to get your API token</h4>
                <ol class="mt-2 space-y-1.5 text-sm text-surface-300">
                  <li class="flex gap-2">
                    <span class="shrink-0 font-mono text-xs text-accent-400">1.</span>
                    <span>Log in to the <a href="https://console.hetzner.cloud" target="_blank" rel="noopener" class="text-accent-400 underline decoration-accent-400/30 hover:decoration-accent-400">Hetzner Cloud Console</a></span>
                  </li>
                  <li class="flex gap-2">
                    <span class="shrink-0 font-mono text-xs text-accent-400">2.</span>
                    <span>Select your project, then go to <strong class="text-surface-100">Security</strong> &rarr; <strong class="text-surface-100">API Tokens</strong></span>
                  </li>
                  <li class="flex gap-2">
                    <span class="shrink-0 font-mono text-xs text-accent-400">3.</span>
                    <span>Click <strong class="text-surface-100">Generate API Token</strong> and select <strong class="text-surface-100">Read &amp; Write</strong></span>
                  </li>
                  <li class="flex gap-2">
                    <span class="shrink-0 font-mono text-xs text-accent-400">4.</span>
                    <span>Copy the token immediately &mdash; it&rsquo;s only shown once</span>
                  </li>
                </ol>
                <a
                  href="https://docs.hetzner.com/cloud/api/getting-started/generating-api-token"
                  target="_blank"
                  rel="noopener"
                  class="mt-2.5 inline-flex items-center gap-1 text-xs text-accent-400 hover:text-accent-300"
                >
                  Official documentation
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                  </svg>
                </a>
              </div>

              <!-- API Token input -->
              <div class="space-y-1.5">
                <Label class="text-sm font-medium text-surface-300">API Token</Label>
                <Input
                  v-model="formToken"
                  type="password"
                  placeholder="Paste your Hetzner API token"
                  :class="inputClass(!!formError)"
                />
                <p v-if="formError" class="text-xs text-danger-400">{{ formError }}</p>
              </div>

              <!-- Validation result feedback -->
              <Transition
                enter-active-class="transition duration-200 ease-out"
                enter-from-class="opacity-0 -translate-y-1"
                enter-to-class="opacity-100 translate-y-0"
                leave-active-class="transition duration-150 ease-in"
                leave-from-class="opacity-100 translate-y-0"
                leave-to-class="opacity-0 -translate-y-1"
              >
                <Alert v-if="formValidation" :class="validationClass(formValidation)">
                  <div class="flex items-start gap-2">
                    <!-- Success icon -->
                    <svg v-if="formValidation.valid && formValidation.read_write" xmlns="http://www.w3.org/2000/svg" class="mt-0.5 h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <!-- Warning icon -->
                    <svg v-else-if="formValidation.valid" xmlns="http://www.w3.org/2000/svg" class="mt-0.5 h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                    </svg>
                    <!-- Error icon -->
                    <svg v-else xmlns="http://www.w3.org/2000/svg" class="mt-0.5 h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <span>{{ formValidation.message }}</span>
                  </div>
                </Alert>
              </Transition>
            </template>

            <!-- Step 2: Name and save -->
            <template v-if="formStep === 'validated'">
              <!-- Success banner -->
              <Alert class="rounded-lg border border-success-600/30 bg-success-900/20 text-sm text-success-400">
                <div class="flex items-center gap-2">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <span>Token verified with Read &amp; Write permissions</span>
                </div>
              </Alert>

              <div class="space-y-1.5">
                <Label class="text-sm font-medium text-surface-300">Connection Name</Label>
                <Input
                  v-model="formName"
                  placeholder="e.g. Production, Staging, My Project"
                  :class="inputClass(!!formError)"
                />
                <p v-if="formError" class="text-xs text-danger-400">{{ formError }}</p>
              </div>
              <p class="text-xs text-surface-500">
                Give this connection a name to identify it later. This is especially useful if you connect multiple Hetzner projects.
              </p>
            </template>
          </div>
        </div>

        <DialogFooter class="border-t border-surface-800/40 px-6 py-4">
          <div class="flex items-center justify-end gap-2">
            <Button variant="ghost" size="sm" class="text-surface-200 hover:text-surface-50" @click="modal.close">
              Cancel
            </Button>

            <Button
              v-if="formStep === 'configure'"
              size="sm"
              class="bg-accent-500 text-surface-950 hover:bg-accent-400"
              :disabled="!formToken.trim() || formValidating"
              @click="handleValidate"
            >
              <Spinner v-if="formValidating" class="text-surface-950" />
              Validate Token
            </Button>

            <Button
              v-if="formStep === 'validated'"
              size="sm"
              class="bg-accent-500 text-surface-950 hover:bg-accent-400"
              :disabled="!formName.trim() || formSaving"
              @click="handleSave"
            >
              <Spinner v-if="formSaving" class="text-surface-950" />
              Save Provider
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
