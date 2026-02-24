<script setup lang="ts">
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

const handleInput = (value: string) => {
  emit('update:modelValue', value)
}
</script>

<template>
  <div class="space-y-1.5">
    <label v-if="props.label" class="block text-sm font-medium text-surface-300">
      {{ props.label }}
    </label>
    <UInput
      :model-value="props.modelValue"
      :placeholder="props.placeholder"
      :type="props.type"
      :disabled="props.disabled"
      :color="props.error ? 'error' : 'primary'"
      :ui="{
        base: props.error 
          ? 'border-danger-500/60 focus:ring-danger-500/40 focus:border-danger-500/60' 
          : 'border-surface-700/60 hover:border-surface-600',
      }"
      @update:model-value="handleInput"
    />
    <p v-if="props.error" class="text-xs text-danger-400">{{ props.error }}</p>
  </div>
</template>
