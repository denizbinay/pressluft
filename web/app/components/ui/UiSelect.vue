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

// Convert options to NuxtUI format - items can be array of { label, value }
const items = computed(() => {
  return props.options.map(option => ({
    label: option.label,
    value: option.value,
  }))
})

const handleUpdate = (value: string) => {
  emit('update:modelValue', value)
}
</script>

<template>
  <div class="space-y-1.5">
    <label v-if="props.label" class="block text-sm font-medium text-surface-300">
      {{ props.label }}
    </label>
    <USelect
      :model-value="props.modelValue"
      :items="items"
      :placeholder="props.placeholder"
      :disabled="props.disabled"
      :ui="{
        base: 'border-surface-700/60 hover:border-surface-600',
      }"
      @update:model-value="handleUpdate"
    />
  </div>
</template>
