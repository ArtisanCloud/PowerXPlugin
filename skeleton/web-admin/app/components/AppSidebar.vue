<template>
  <aside
    class="w-64 min-w-64 max-w-64 bg-white dark:bg-gray-900 border-r border-gray-200 dark:border-gray-800 min-h-screen flex-shrink-0"
  >
    <nav class="p-4 space-y-6">
      <div>
        <UButton
          to="/intro"
          variant="ghost"
          color="neutral"
          class="w-full justify-start"
          :class="{
            'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400':
              isExactActive('/intro') || isExactActive('/'),
          }"
        >
          <UIcon name="i-heroicons-information-circle" class="w-4 h-4 mr-3" />
          {{ t('navigation.intro') }}
        </UButton>
      </div>

      <div>
        <div class="px-3 py-2 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
          {{ t('navigation.templates') }}
        </div>
        <div class="space-y-1">
          <UButton
            variant="ghost"
            color="neutral"
            class="w-full justify-start"
            @click="toggleTemplatesMenu"
            :class="{
              'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400':
                isGroupActive(['/templates']),
            }"
          >
            <UIcon name="i-heroicons-clipboard-document-list" class="w-4 h-4 mr-3" />
            {{ t('navigation.templates') }}
            <UIcon
              :name="showTemplatesMenu ? 'i-heroicons-chevron-down' : 'i-heroicons-chevron-right'"
              class="w-4 h-4 ml-auto"
            />
          </UButton>

          <div v-show="showTemplatesMenu" class="ml-6 mt-1 space-y-1">
            <UButton
              to="/templates"
              variant="ghost"
              color="neutral"
              size="sm"
              class="w-full justify-start text-sm"
              :class="{
                'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400':
                  isExactActive('/templates'),
              }"
            >
              <UIcon name="i-heroicons-document-text" class="w-3 h-3 mr-2" />
              {{ t('templates.overview.title') }}
            </UButton>
            <UButton
              to="/templates/develop"
              variant="ghost"
              color="neutral"
              size="sm"
              class="w-full justify-start text-sm"
              :class="{
                'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400':
                  isExactActive('/templates/develop'),
              }"
            >
              <UIcon name="i-heroicons-cpu-chip" class="w-3 h-3 mr-2" />
              {{ t('navigation.templatesDevelop') }}
            </UButton>
            <UButton
              to="/templates/crud"
              variant="ghost"
              color="neutral"
              size="sm"
              class="w-full justify-start text-sm"
              :class="{
                'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400':
                  isExactActive('/templates/crud'),
              }"
            >
              <UIcon name="i-heroicons-wrench" class="w-3 h-3 mr-2" />
              {{ t('navigation.templatesCrud') }}
            </UButton>
          </div>
        </div>
      </div>

      
    </nav>
  </aside>
</template>

<script setup lang="ts">
const { t } = useI18n();
const route = useRoute();

const showTemplatesMenu = ref(true);

const normalizePath = (value: string) => {
  if (!value) {
    return "/";
  }
  if (value !== "/" && value.endsWith("/")) {
    return value.replace(/\/+$/, "");
  }
  return value;
};

const isExactActive = (target: string) => {
  const current = normalizePath(route.path);
  const normalizedTarget = normalizePath(target);

  if (normalizedTarget === "/") {
    return current === "/";
  }

  return current === normalizedTarget;
};

const isGroupActive = (prefixes: string | string[]) => {
  const current = normalizePath(route.path);
  const list = Array.isArray(prefixes) ? prefixes : [prefixes];

  return list.some((prefix) => {
    const normalizedPrefix = normalizePath(prefix);
    return (
      current === normalizedPrefix || current.startsWith(`${normalizedPrefix}/`)
    );
  });
};

const toggleTemplatesMenu = () => {
  showTemplatesMenu.value = !showTemplatesMenu.value;
};

watch(
  () => route.path,
  (newPath) => {
    if (newPath.startsWith("/templates")) {
      showTemplatesMenu.value = true;
    }
  }
);
</script>
