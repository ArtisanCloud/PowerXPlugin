<template>
  <UModal v-model="visible" :prevent-close="loading">
    <UCard>
      <template #header>
        <h3 class="text-lg font-medium text-gray-900 dark:text-white">
          {{ title }}
        </h3>
      </template>

      <p class="text-gray-600 dark:text-gray-300">
        {{ message }}
      </p>

      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton color="gray" variant="ghost" :disabled="loading" @click="cancel">
            {{ $t('common.cancel') }}
          </UButton>
          <UButton :color="confirmColor" :loading="loading" @click="confirm">
            {{ confirmText || $t('common.confirm') }}
          </UButton>
        </div>
      </template>
    </UCard>
  </UModal>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{
  modelValue: boolean
  title: string
  message: string
  confirmText?: string
  confirmColor?: string
  loading?: boolean
}>(), {
  confirmColor: 'primary'
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'confirm'): void
  (e: 'cancel'): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value)
})

function cancel() {
  emit('cancel')
  visible.value = false
}

function confirm() {
  emit('confirm')
}
</script>
