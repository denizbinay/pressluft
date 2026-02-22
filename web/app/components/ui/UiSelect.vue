<script setup lang="ts">
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

const handleChange = (event: Event) => {
  const target = event.target as HTMLSelectElement
  emit('update:modelValue', target.value)
}
</script>

<template>
  <div class="space-y-1.5">
    <label v-if="props.label" class="block text-sm font-medium text-surface-300">
      {{ props.label }}
    </label>
    <select
      :value="props.modelValue"
      :disabled="props.disabled"
      :class="[
        'w-full appearance-none rounded-lg border border-surface-700/60 bg-surface-900/50 px-3.5 py-2 pr-10 text-sm text-surface-100',
        'transition-colors duration-150',
        'focus:outline-none focus:ring-2 focus:ring-accent-500/40 focus:border-accent-500/60',
        'hover:border-surface-600',
        'disabled:opacity-40 disabled:cursor-not-allowed',
      ]"
      @change="handleChange"
    >
      <option value="" disabled class="text-surface-500">{{ props.placeholder }}</option>
      <option
        v-for="option in props.options"
        :key="option.value"
        :value="option.value"
        class="bg-surface-900 text-surface-100"
      >
        {{ option.label }}
      </option>
    </select>
  </div>
</template>
