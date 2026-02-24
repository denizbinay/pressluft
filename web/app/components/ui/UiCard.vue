<script setup lang="ts">
interface Props {
  hoverable?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  hoverable: false,
})

// Build ui prop for UCard to maintain dark theme styling
const cardUi = computed(() => ({
  root: [
    'rounded-xl border border-surface-800/60 bg-surface-900/50 backdrop-blur-sm',
    props.hoverable && 'transition-all duration-200 hover:border-surface-700/80 hover:bg-surface-900/70 hover:shadow-lg hover:shadow-surface-950/50 cursor-pointer',
  ],
  header: 'border-b border-surface-800/40 px-5 py-4',
  body: 'px-5 py-4',
  footer: 'border-t border-surface-800/40 px-5 py-3',
}))
</script>

<template>
  <UCard :ui="cardUi">
    <template v-if="$slots.header" #header>
      <slot name="header" />
    </template>
    
    <slot />
    
    <template v-if="$slots.footer" #footer>
      <slot name="footer" />
    </template>
  </UCard>
</template>
