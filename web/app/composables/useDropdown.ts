import { ref, readonly, onMounted, onBeforeUnmount } from 'vue'

export function useDropdown() {
  const isOpen = ref(false)
  const triggerRef = ref<HTMLElement | null>(null)
  const menuRef = ref<HTMLElement | null>(null)

  const open = () => { isOpen.value = true }
  const close = () => { isOpen.value = false }
  const toggle = () => { isOpen.value = !isOpen.value }

  const handleClickOutside = (event: MouseEvent) => {
    const target = event.target as Node
    const isInsideTrigger = triggerRef.value?.contains(target) ?? false
    const isInsideMenu = menuRef.value?.contains(target) ?? false

    if (!isInsideTrigger && !isInsideMenu) {
      close()
    }
  }

  const handleEscape = (event: KeyboardEvent) => {
    if (event.key === 'Escape') {
      close()
    }
  }

  onMounted(() => {
    document.addEventListener('click', handleClickOutside)
    document.addEventListener('keydown', handleEscape)
  })

  onBeforeUnmount(() => {
    document.removeEventListener('click', handleClickOutside)
    document.removeEventListener('keydown', handleEscape)
  })

  return {
    isOpen: readonly(isOpen),
    triggerRef,
    menuRef,
    open,
    close,
    toggle,
  }
}
