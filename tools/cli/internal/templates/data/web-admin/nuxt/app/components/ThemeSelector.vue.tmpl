<template>
  <UDropdownMenu :items="dropdownItems" :popper="{ placement: 'bottom-end' }">
    <UButton
      color="neutral"
      variant="ghost"
      :icon="currentThemeIcon"
      :aria-label="t('theme.select')"
      data-testid="admin-theme"
      size="sm"
      square
    />
  </UDropdownMenu>
</template>

<script setup lang="ts">
const { theme, setTheme } = useTheme();
const { t } = useI18n();

// 主题图标映射
const themeIcons = {
  light: "i-heroicons-sun",
  dark: "i-heroicons-moon",
  system: "i-heroicons-computer-desktop",
};

const currentThemeIcon = computed(() => {
  return (
    themeIcons[theme.value as keyof typeof themeIcons] || "i-heroicons-sun"
  );
});

const dropdownItems = computed(() => [
  (["light", "dark", "system"] as const).map((value) => ({
    label: t(`theme.${value}`),
    icon: themeIcons[value],
    onSelect: () => {
      setTheme(value);
    },
  })),
]);
</script>
