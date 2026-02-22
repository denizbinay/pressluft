<script setup lang="ts">
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

const toggle = () => {
  if (!props.disabled) {
    emit('update:modelValue', !props.modelValue)
  }
}
</script>

<template>
  <label
    :class="[
      'inline-flex items-center gap-3 cursor-pointer select-none',
      props.disabled && 'opacity-40 cursor-not-allowed',
    ]"
  >
    <button
      type="button"
      role="switch"
      :aria-checked="props.modelValue"
      :disabled="props.disabled"
      :class="[
        'relative inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors duration-200',
        props.modelValue ? 'bg-accent-500' : 'bg-surface-700',
      ]"
      @click="toggle"
    >
      <span
        :class="[
          'pointer-events-none inline-block h-3.5 w-3.5 rounded-full bg-white shadow-sm transition-transform duration-200',
          props.modelValue ? 'translate-x-[18px]' : 'translate-x-[3px]',
        ]"
      />
    </button>
    <span v-if="props.label" class="text-sm text-surface-300">{{ props.label }}</span>
  </label>
</template>
