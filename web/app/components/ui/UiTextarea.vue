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

const handleInput = (event: Event) => {
  const target = event.target as HTMLTextAreaElement
  emit('update:modelValue', target.value)
}
</script>

<template>
  <div class="space-y-1.5">
    <label v-if="props.label" class="block text-sm font-medium text-surface-300">
      {{ props.label }}
    </label>
    <textarea
      :value="props.modelValue"
      :placeholder="props.placeholder"
      :rows="props.rows"
      :disabled="props.disabled"
      :class="[
        'w-full resize-none rounded-lg border border-surface-700/60 bg-surface-900/50 px-3.5 py-2.5 text-sm text-surface-100 placeholder-surface-500',
        'transition-colors duration-150',
        'focus:outline-none focus:ring-2 focus:ring-accent-500/40 focus:border-accent-500/60',
        'hover:border-surface-600',
        'disabled:opacity-40 disabled:cursor-not-allowed',
      ]"
      @input="handleInput"
    />
  </div>
</template>
