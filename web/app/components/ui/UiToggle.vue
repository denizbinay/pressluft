<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  modelValue?: boolean
  label?: string
  disabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: false,
  label: '',
  disabled: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const toggleClass = computed(() => [
  'relative inline-flex h-6 w-11 items-center rounded-full border transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-500/60 focus-visible:ring-offset-2 focus-visible:ring-offset-surface-950',
  props.modelValue ? 'border-accent-500/60 bg-accent-500/40' : 'border-surface-700/60 bg-surface-800/60',
  props.disabled && 'cursor-not-allowed opacity-60',
])

const knobClass = computed(() => [
  'inline-block h-5 w-5 transform rounded-full bg-surface-100 shadow-sm transition',
  props.modelValue ? 'translate-x-5' : 'translate-x-1',
])

const handleUpdate = () => {
  if (!props.disabled) {
    emit('update:modelValue', !props.modelValue)
  }
}
</script>

<template>
  <div class="flex items-center justify-between gap-3">
    <label
      v-if="props.label"
      class="text-sm text-surface-200"
      :class="!props.disabled && 'cursor-pointer'"
      @click="handleUpdate"
    >
      {{ props.label }}
    </label>
    <button
      type="button"
      role="switch"
      :aria-checked="props.modelValue"
      :disabled="props.disabled"
      :class="toggleClass"
      @click="handleUpdate"
    >
      <span :class="knobClass" />
    </button>
  </div>
</template>
