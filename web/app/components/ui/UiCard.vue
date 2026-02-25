<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  hoverable?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  hoverable: false,
})

const cardClass = computed(() => [
  'rounded-xl border border-surface-800/60 bg-surface-900/50 backdrop-blur-sm',
  props.hoverable && 'transition-all duration-200 hover:border-surface-700/80 hover:bg-surface-900/70 hover:shadow-lg hover:shadow-surface-950/50 cursor-pointer',
])
</script>

<template>
  <section :class="cardClass">
    <header v-if="$slots.header" class="border-b border-surface-800/40 px-6 py-5">
      <slot name="header" />
    </header>

    <div class="px-6 py-5">
      <slot />
    </div>

    <footer v-if="$slots.footer" class="border-t border-surface-800/40 px-6 py-4">
      <slot name="footer" />
    </footer>
  </section>
</template>
