<script setup lang="ts">
import { computed } from 'vue'

interface SelectOption {
  label: string
  value: string
}

interface Props {
  modelValue?: string
  label?: string
  options: SelectOption[]
  placeholder?: string
  disabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  label: '',
  placeholder: 'Select an option',
  disabled: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const selectClass = computed(() => [
  'w-full appearance-none rounded-lg border bg-surface-900/60 px-3 py-2 text-sm text-surface-100 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-500/60 focus-visible:ring-offset-2 focus-visible:ring-offset-surface-950',
  'border-surface-700/60 hover:border-surface-600',
  props.disabled && 'cursor-not-allowed opacity-60',
])

const handleUpdate = (event: Event) => {
  const target = event.target as HTMLSelectElement
  emit('update:modelValue', target.value)
}
</script>

<template>
  <div class="space-y-1.5">
    <label v-if="props.label" class="block text-sm font-medium text-surface-300">
      {{ props.label }}
    </label>
    <div class="relative">
      <select
        :value="props.modelValue"
        :disabled="props.disabled"
        :class="selectClass"
        @change="handleUpdate"
      >
        <option value="" disabled>
          {{ props.placeholder }}
        </option>
        <option v-for="option in props.options" :key="option.value" :value="option.value">
          {{ option.label }}
        </option>
      </select>
      <span class="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-surface-400">
        <svg viewBox="0 0 20 20" fill="currentColor" class="h-4 w-4" aria-hidden="true">
          <path
            fill-rule="evenodd"
            d="M5.23 7.21a.75.75 0 0 1 1.06.02L10 10.94l3.71-3.71a.75.75 0 1 1 1.06 1.06l-4.24 4.25a.75.75 0 0 1-1.06 0L5.21 8.29a.75.75 0 0 1 .02-1.08Z"
            clip-rule="evenodd"
          />
        </svg>
      </span>
    </div>
  </div>
</template>
