<template>
  <UModal
    v-model:open="open"
    :title="dialogTitle"
    :description="resolvedDescription || undefined"
    :ui="{
      content: 'w-full max-w-md space-y-4'
    }"
  >
    <template #body>
      <slot>
        <p
          v-if="showBodyMessage"
          class="text-sm text-gray-600 dark:text-gray-300"
        >
          {{ bodyMessage }}
        </p>
      </slot>
    </template>

    <template #footer>
      <div class="ml-auto flex w-full max-w-full items-center justify-end gap-3">
        <UButton
          variant="ghost"
          :disabled="loading"
          @click="handleCancel"
        >
          {{ cancelText }}
        </UButton>
        <UButton
          :color="buttonColor"
          :loading="loading"
          @click="handleConfirm"
        >
          {{ confirmText }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { computed, useSlots } from "vue"
import { useI18n } from "vue-i18n"

const props = withDefaults(defineProps<{
  modelValue?: boolean
  title?: string
  description?: string
  message?: string
  confirmText?: string
  cancelText?: string
  confirmColor?: "primary" | "secondary" | "success" | "error" | "warning" | "info" | "neutral"
  loading?: boolean
}>(), {
  modelValue: false,
  confirmColor: "primary",
  loading: false,
})

const emit = defineEmits<{
  (e: "update:modelValue", value: boolean): void
  (e: "confirm"): void
  (e: "cancel"): void
}>()

const { t } = useI18n()
const slots = useSlots()

const open = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit("update:modelValue", value),
})

const dialogTitle = computed(
  () => props.title || t("common.confirmation")
)

const confirmText = computed(
  () => props.confirmText || t("common.confirm")
)

const cancelText = computed(
  () => props.cancelText || t("common.cancel")
)

const loading = computed(() => props.loading)

const buttonColor = computed(() => props.confirmColor)

const resolvedDescription = computed(() =>
  props.description ?? ""
)

const bodyMessage = computed(() =>
  props.message ?? ""
)

const showBodyMessage = computed(() =>
  !slots.default && !!bodyMessage.value
)

const handleCancel = () => {
  emit("cancel")
  open.value = false
}

const handleConfirm = () => {
  emit("confirm")
}
</script>
