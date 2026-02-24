<script setup lang="ts">
interface Props {
  modelValue?: string
  label?: string
  placeholder?: string
  rows?: number
  disabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  label: '',
  placeholder: '',
  rows: 4,
  disabled: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const handleInput = (value: string) => {
  emit('update:modelValue', value)
}
</script>

<template>
  <div class="space-y-1.5">
    <label v-if="props.label" class="block text-sm font-medium text-surface-300">
      {{ props.label }}
    </label>
    <UTextarea
      :model-value="props.modelValue"
      :placeholder="props.placeholder"
      :rows="props.rows"
      :disabled="props.disabled"
      :ui="{
        base: 'border-surface-700/60 hover:border-surface-600',
      }"
      @update:model-value="handleInput"
    />
  </div>
</template>
