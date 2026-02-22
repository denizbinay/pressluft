<script setup lang="ts">
interface Props {
  modelValue?: string
  label?: string
  placeholder?: string
  type?: string
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
      :type="props.type"
      :value="props.modelValue"
      :placeholder="props.placeholder"
      :disabled="props.disabled"
      :class="[
        'w-full rounded-lg border bg-surface-900/50 px-3.5 py-2 text-sm text-surface-100 placeholder-surface-500',
        'transition-colors duration-150',
        'focus:outline-none focus:ring-2 focus:ring-accent-500/40 focus:border-accent-500/60',
        'disabled:opacity-40 disabled:cursor-not-allowed',
        props.error
          ? 'border-danger-500/60 focus:ring-danger-500/40 focus:border-danger-500/60'
          : 'border-surface-700/60 hover:border-surface-600',
      ]"
      @input="handleInput"
    />
    <p v-if="props.error" class="text-xs text-danger-400">{{ props.error }}</p>
  </div>
</template>
