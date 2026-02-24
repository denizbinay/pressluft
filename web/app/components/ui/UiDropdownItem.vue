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
        ? 'text-error hover:bg-error/10 hover:text-error'
        : 'text-foreground hover:bg-default focus:bg-default',
      props.disabled && 'opacity-40 pointer-events-none',
    ]"
    :disabled="props.disabled"
  >
    <slot />
  </button>
</template>
