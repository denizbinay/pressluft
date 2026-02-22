<script setup lang="ts">
// ── Reactive state for interactive demos ──
const inputValue = ref('')
const textareaValue = ref('')
const selectValue = ref('')
const toggleA = ref(true)
const toggleB = ref(false)
const loadingBtn = ref(false)

const selectOptions = [
  { label: 'us-east-1', value: 'us-east-1' },
  { label: 'eu-west-1', value: 'eu-west-1' },
  { label: 'ap-southeast-1', value: 'ap-southeast-1' },
]

// Modal
const { isOpen: modalOpen, open: openModal, close: closeModal } = useModal()

// Simulate loading
const simulateLoading = () => {
  loadingBtn.value = true
  setTimeout(() => { loadingBtn.value = false }, 2000)
}
</script>

<template>
  <div class="space-y-12">
    <!-- Page header -->
    <div>
      <h1 class="text-2xl font-semibold text-surface-50">
        UI Components
      </h1>
      <p class="mt-2 text-sm text-surface-400">
        Component library &mdash; your design system at a glance.
      </p>
    </div>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Stats row -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <UiCard v-for="stat in [
        { label: 'Deployments', value: '1,284', change: '+12%', variant: 'success' as const },
        { label: 'Active Services', value: '23', change: '+2', variant: 'info' as const },
        { label: 'Avg Response', value: '142ms', change: '-8ms', variant: 'success' as const },
        { label: 'Error Rate', value: '0.03%', change: '+0.01%', variant: 'warning' as const },
      ]" :key="stat.label" hoverable>
        <div class="flex items-start justify-between">
          <div>
            <p class="text-xs font-medium uppercase tracking-wider text-surface-500">
              {{ stat.label }}
            </p>
            <p class="mt-1 text-2xl font-bold text-surface-50 font-mono">
              {{ stat.value }}
            </p>
          </div>
          <UiBadge :variant="stat.variant">{{ stat.change }}</UiBadge>
        </div>
      </UiCard>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Buttons -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-4 text-lg font-semibold text-surface-100">Buttons</h2>
      <UiCard>
        <div class="space-y-4">
          <!-- Variants -->
          <div>
            <p class="mb-3 text-xs font-medium uppercase tracking-wider text-surface-500">Variants</p>
            <div class="flex flex-wrap items-center gap-3">
              <UiButton variant="primary">Primary</UiButton>
              <UiButton variant="secondary">Secondary</UiButton>
              <UiButton variant="outline">Outline</UiButton>
              <UiButton variant="ghost">Ghost</UiButton>
              <UiButton variant="danger">Danger</UiButton>
            </div>
          </div>

          <!-- Sizes -->
          <div>
            <p class="mb-3 text-xs font-medium uppercase tracking-wider text-surface-500">Sizes</p>
            <div class="flex flex-wrap items-center gap-3">
              <UiButton size="sm">Small</UiButton>
              <UiButton size="md">Medium</UiButton>
              <UiButton size="lg">Large</UiButton>
            </div>
          </div>

          <!-- States -->
          <div>
            <p class="mb-3 text-xs font-medium uppercase tracking-wider text-surface-500">States</p>
            <div class="flex flex-wrap items-center gap-3">
              <UiButton disabled>Disabled</UiButton>
              <UiButton :loading="loadingBtn" @click="simulateLoading">
                {{ loadingBtn ? 'Deploying...' : 'Click to load' }}
              </UiButton>
            </div>
          </div>
        </div>
      </UiCard>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Badges -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-4 text-lg font-semibold text-surface-100">Badges</h2>
      <UiCard>
        <div class="flex flex-wrap items-center gap-3">
          <UiBadge>Default</UiBadge>
          <UiBadge variant="success">Healthy</UiBadge>
          <UiBadge variant="warning">Degraded</UiBadge>
          <UiBadge variant="danger">Down</UiBadge>
          <UiBadge variant="info">Deploying</UiBadge>
        </div>
      </UiCard>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Progress Bars -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-4 text-lg font-semibold text-surface-100">Progress Bars</h2>
      <UiCard>
        <div class="space-y-5">
          <UiProgressBar :value="72" show-label>
            <template #label>
              <span class="text-xs font-medium text-surface-300">CPU Usage</span>
            </template>
          </UiProgressBar>
          <UiProgressBar :value="45" variant="success" show-label>
            <template #label>
              <span class="text-xs font-medium text-surface-300">Memory</span>
            </template>
          </UiProgressBar>
          <UiProgressBar :value="89" variant="warning" show-label>
            <template #label>
              <span class="text-xs font-medium text-surface-300">Disk</span>
            </template>
          </UiProgressBar>
          <UiProgressBar :value="12" variant="danger" size="sm" />
          <UiProgressBar :value="60" size="lg" />
        </div>
      </UiCard>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Cards -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-4 text-lg font-semibold text-surface-100">Cards</h2>
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <UiCard>
          <template #header>
            <h3 class="text-sm font-semibold text-surface-100">Basic Card</h3>
          </template>
          <p class="text-sm text-surface-400">
            A standard card with header, body, and footer slots. Use it for grouping related content.
          </p>
          <template #footer>
            <div class="flex justify-end">
              <UiButton size="sm" variant="ghost">View details</UiButton>
            </div>
          </template>
        </UiCard>

        <UiCard hoverable>
          <template #header>
            <div class="flex items-center justify-between">
              <h3 class="text-sm font-semibold text-surface-100">Hoverable Card</h3>
              <UiBadge variant="success">Active</UiBadge>
            </div>
          </template>
          <p class="text-sm text-surface-400">
            This card has a hover effect. Useful for clickable items like service cards or pipeline entries.
          </p>
        </UiCard>
      </div>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Forms -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-4 text-lg font-semibold text-surface-100">Form Controls</h2>
      <UiCard>
        <div class="grid grid-cols-1 gap-5 md:grid-cols-2">
          <UiInput
            v-model="inputValue"
            label="Service Name"
            placeholder="e.g. api-gateway"
          />
          <UiSelect
            v-model="selectValue"
            label="Region"
            :options="selectOptions"
          />
          <div class="md:col-span-2">
            <UiTextarea
              v-model="textareaValue"
              label="Description"
              placeholder="Describe the service configuration..."
            />
          </div>
          <UiInput
            label="With Error"
            placeholder="Invalid input"
            error="This field is required"
            model-value="bad-value"
          />
          <UiInput
            label="Disabled"
            placeholder="Cannot edit"
            disabled
          />
        </div>
      </UiCard>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Toggles -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-4 text-lg font-semibold text-surface-100">Toggles</h2>
      <UiCard>
        <div class="space-y-4">
          <UiToggle v-model="toggleA" label="Auto-deploy on push" />
          <UiToggle v-model="toggleB" label="Enable notifications" />
          <UiToggle label="Disabled toggle" disabled />
        </div>
      </UiCard>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Dropdown -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-4 text-lg font-semibold text-surface-100">Dropdown</h2>
      <UiCard>
        <div class="flex gap-4">
          <UiDropdown>
            <template #trigger>
              <UiButton variant="secondary">
                Actions
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
                </svg>
              </UiButton>
            </template>
            <UiDropdownItem>Deploy</UiDropdownItem>
            <UiDropdownItem>Rollback</UiDropdownItem>
            <UiDropdownItem>View Logs</UiDropdownItem>
            <div class="my-1 border-t border-surface-700/40" />
            <UiDropdownItem danger>Delete Service</UiDropdownItem>
          </UiDropdown>

          <UiDropdown align="right">
            <template #trigger>
              <UiButton variant="outline">
                Options
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
                </svg>
              </UiButton>
            </template>
            <UiDropdownItem>Edit</UiDropdownItem>
            <UiDropdownItem>Duplicate</UiDropdownItem>
            <UiDropdownItem disabled>Archive</UiDropdownItem>
          </UiDropdown>
        </div>
      </UiCard>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Modal -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-4 text-lg font-semibold text-surface-100">Modal</h2>
      <UiCard>
        <UiButton @click="openModal">Open Modal</UiButton>
      </UiCard>

      <UiModal :open="modalOpen" title="Confirm Deployment" @close="closeModal">
        <p class="text-sm text-surface-400">
          You are about to deploy <span class="font-mono text-surface-200">api-gateway@v2.4.1</span>
          to <span class="font-semibold text-surface-200">production</span>. This action will
          replace the currently running version.
        </p>
        <template #footer>
          <div class="flex justify-end gap-3">
            <UiButton variant="ghost" @click="closeModal">Cancel</UiButton>
            <UiButton @click="closeModal">Deploy</UiButton>
          </div>
        </template>
      </UiModal>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Typography & Colors -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-4 text-lg font-semibold text-surface-100">Typography &amp; Colors</h2>
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <UiCard>
          <template #header>
            <h3 class="text-sm font-semibold text-surface-100">Type Scale</h3>
          </template>
          <div class="space-y-3">
            <p class="text-3xl font-bold text-surface-50">Heading 1</p>
            <p class="text-2xl font-semibold text-surface-50">Heading 2</p>
            <p class="text-xl font-semibold text-surface-100">Heading 3</p>
            <p class="text-base text-surface-200">Body text — Inter 400</p>
            <p class="text-sm text-surface-400">Secondary text — Inter 400</p>
            <p class="text-xs text-surface-500">Caption text — Inter 400</p>
            <p class="font-mono text-sm text-accent-400">Monospace — JetBrains Mono</p>
          </div>
        </UiCard>

        <UiCard>
          <template #header>
            <h3 class="text-sm font-semibold text-surface-100">Color Palette</h3>
          </template>
          <div class="space-y-3">
            <div>
              <p class="mb-1.5 text-xs font-medium text-surface-500">Surface</p>
              <div class="flex gap-1">
                <div class="h-8 w-8 rounded bg-surface-950" title="950" />
                <div class="h-8 w-8 rounded bg-surface-900" title="900" />
                <div class="h-8 w-8 rounded bg-surface-800" title="800" />
                <div class="h-8 w-8 rounded bg-surface-700" title="700" />
                <div class="h-8 w-8 rounded bg-surface-600" title="600" />
                <div class="h-8 w-8 rounded bg-surface-500" title="500" />
                <div class="h-8 w-8 rounded bg-surface-400" title="400" />
                <div class="h-8 w-8 rounded bg-surface-300" title="300" />
              </div>
            </div>
            <div>
              <p class="mb-1.5 text-xs font-medium text-surface-500">Accent</p>
              <div class="flex gap-1">
                <div class="h-8 w-8 rounded bg-accent-700" title="700" />
                <div class="h-8 w-8 rounded bg-accent-600" title="600" />
                <div class="h-8 w-8 rounded bg-accent-500" title="500" />
                <div class="h-8 w-8 rounded bg-accent-400" title="400" />
                <div class="h-8 w-8 rounded bg-accent-300" title="300" />
              </div>
            </div>
            <div>
              <p class="mb-1.5 text-xs font-medium text-surface-500">Semantic</p>
              <div class="flex gap-2">
                <div class="flex items-center gap-1.5">
                  <div class="h-6 w-6 rounded bg-success-500" />
                  <span class="text-xs text-surface-400">Success</span>
                </div>
                <div class="flex items-center gap-1.5">
                  <div class="h-6 w-6 rounded bg-warning-500" />
                  <span class="text-xs text-surface-400">Warning</span>
                </div>
                <div class="flex items-center gap-1.5">
                  <div class="h-6 w-6 rounded bg-danger-500" />
                  <span class="text-xs text-surface-400">Danger</span>
                </div>
                <div class="flex items-center gap-1.5">
                  <div class="h-6 w-6 rounded bg-primary-500" />
                  <span class="text-xs text-surface-400">Primary</span>
                </div>
              </div>
            </div>
          </div>
        </UiCard>
      </div>
    </section>

    <!-- ────────────────────────────────────────────────────────────── -->
    <!-- Glass effect demo -->
    <!-- ────────────────────────────────────────────────────────────── -->
    <section>
      <h2 class="mb-4 text-lg font-semibold text-surface-100">Special Effects</h2>
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div class="glass rounded-xl p-5">
          <h3 class="text-sm font-semibold text-surface-100">Glass Effect</h3>
          <p class="mt-1 text-sm text-surface-400">
            Frosted glass panel with backdrop blur. Use for overlays and elevated surfaces.
          </p>
        </div>
        <div class="glow-accent rounded-xl border border-accent-500/20 bg-surface-900/50 p-5">
          <h3 class="text-sm font-semibold text-accent-300">Accent Glow</h3>
          <p class="mt-1 text-sm text-surface-400">
            Subtle glow effect for highlighted or featured elements.
          </p>
        </div>
      </div>
    </section>
  </div>
</template>
