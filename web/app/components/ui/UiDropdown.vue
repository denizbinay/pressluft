<script setup lang="ts">
interface Props {
  align?: 'left' | 'right'
}

const props = withDefaults(defineProps<Props>(), {
  align: 'left',
})

const { isOpen, triggerRef, menuRef, toggle } = useDropdown()
</script>

<template>
  <div class="relative inline-block">
    <!-- Trigger -->
    <div ref="triggerRef" @click="toggle">
      <slot name="trigger" />
    </div>

    <!-- Menu -->
    <Transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="opacity-0 scale-95 -translate-y-1"
      enter-to-class="opacity-100 scale-100 translate-y-0"
      leave-active-class="transition duration-100 ease-in"
      leave-from-class="opacity-100 scale-100 translate-y-0"
      leave-to-class="opacity-0 scale-95 -translate-y-1"
    >
      <div
        v-if="isOpen"
        ref="menuRef"
        :class="[
          'absolute z-50 mt-1.5 min-w-[12rem] origin-top rounded-lg border border-surface-700/60 bg-surface-850 p-1 shadow-xl shadow-surface-950/50',
          props.align === 'right' ? 'right-0' : 'left-0',
        ]"
      >
        <slot />
      </div>
    </Transition>
  </div>
</template>
