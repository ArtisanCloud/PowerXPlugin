<template>
  <UToast v-model="visible" :timeout="duration">
    <template #title>
      <span class="font-medium">{{ title }}</span>
    </template>
    <span>{{ message }}</span>
  </UToast>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{
  modelValue: boolean
  title: string
  message: string
  duration?: number
}>(), {
  duration: 3000
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value)
})
</script>
