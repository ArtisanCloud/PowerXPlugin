<template>
  <UContainer class="py-12">
    <div class="text-center space-y-4">
      <h1 class="text-3xl font-semibold">
        {{ $t("intro.title") }}
      </h1>
      <p class="text-gray-500">
        {{ $t("intro.description") }}
      </p>
      <UButton to="/intro" size="lg" color="primary">
        {{ $t("intro.viewIntro") }}
      </UButton>
    </div>
  </UContainer>
</template>

<script setup lang="ts">
definePageMeta({
  title: "PowerXPluginIntro",
})
</script>
