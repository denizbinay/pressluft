<script setup lang="ts">
interface Props {
  disabled?: boolean
  danger?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  danger: false,
})
</script>

<template>
  <button
    :class="[
      'flex w-full items-center gap-2 rounded-md px-3 py-1.5 text-left text-sm transition-colors',
      props.danger
        ? 'text-danger-400 hover:bg-danger-900/40 hover:text-danger-300'
        : 'text-surface-300 hover:bg-surface-800 hover:text-surface-100',
      props.disabled && 'opacity-40 pointer-events-none',
    ]"
    :disabled="props.disabled"
  >
    <slot />
  </button>
</template>
