<template>
  <UDropdownMenu :items="dropdownItems" :popper="{ placement: 'bottom-end' }">
    <UButton
      color="neutral"
      variant="ghost"
      :label="currentLocale?.name || t('theme.language')"
      :aria-label="t('theme.language')"
      data-testid="admin-language"
      trailing-icon="i-heroicons-chevron-down-20-solid"
      size="sm"
    />
  </UDropdownMenu>
</template>

<script setup lang="ts">
const { locale, locales, setLocale, t } = useI18n();

const currentLocale = computed(() => {
  return locales.value.find((l) => l.code === locale.value) || locales.value[0];
});

const dropdownItems = computed(() => [
  locales.value.map((localeOption) => ({
    label: localeOption.name,
    onSelect: () => {
      setLocale(localeOption.code);
    },
  })),
]);
</script>
