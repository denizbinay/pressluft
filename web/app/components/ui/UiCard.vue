<script setup lang="ts">
interface Props {
  hoverable?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  hoverable: false,
})
</script>

<template>
  <div
    :class="[
      'rounded-xl border border-surface-800/60 bg-surface-900/50 backdrop-blur-sm',
      props.hoverable && 'transition-all duration-200 hover:border-surface-700/80 hover:bg-surface-900/70 hover:shadow-lg hover:shadow-surface-950/50',
    ]"
  >
    <div v-if="$slots.header" class="border-b border-surface-800/40 px-5 py-4">
      <slot name="header" />
    </div>
    <div class="px-5 py-4">
      <slot />
    </div>
    <div v-if="$slots.footer" class="border-t border-surface-800/40 px-5 py-3">
      <slot name="footer" />
    </div>
  </div>
</template>
