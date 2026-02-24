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

const handleUpdate = (value: boolean) => {
  if (!props.disabled) {
    emit('update:modelValue', value)
  }
}
</script>

<template>
  <USwitch
    :model-value="props.modelValue"
    :label="props.label"
    :disabled="props.disabled"
    color="primary"
    @update:model-value="handleUpdate"
  />
</template>
