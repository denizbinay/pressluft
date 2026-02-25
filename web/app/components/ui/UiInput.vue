<script setup lang="ts">
import { computed, useAttrs } from 'vue'

interface Props {
  modelValue?: string
  label?: string
  placeholder?: string
  type?: 'text' | 'email' | 'password' | 'search' | 'tel' | 'url' | 'number'
  disabled?: boolean
  error?: string
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  label: '',
  placeholder: '',
  type: 'text',
  disabled: false,
  error: '',
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const attrs = useAttrs()

const inputClass = computed(() => [
  'w-full rounded-lg border bg-surface-900/60 px-3 py-2 text-sm text-surface-100 placeholder:text-surface-400 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-surface-950',
  props.error
    ? 'border-danger-500/60 focus-visible:ring-danger-500/40'
    : 'border-surface-700/60 focus-visible:ring-accent-500/60 hover:border-surface-600',
  props.disabled && 'cursor-not-allowed opacity-60',
])

const handleInput = (event: Event) => {
  const target = event.target as HTMLInputElement
  emit('update:modelValue', target.value)
}
</script>

<template>
  <div class="space-y-1.5">
    <label v-if="props.label" class="block text-sm font-medium text-surface-300">
      {{ props.label }}
    </label>
    <input
      v-bind="attrs"
      :value="props.modelValue"
      :placeholder="props.placeholder"
      :type="props.type"
      :disabled="props.disabled"
      :class="inputClass"
      @input="handleInput"
    />
    <p v-if="props.error" class="text-xs text-danger-400">{{ props.error }}</p>
  </div>
</template>
