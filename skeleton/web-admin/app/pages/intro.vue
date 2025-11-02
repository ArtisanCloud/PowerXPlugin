<template>
  <UContainer class="py-12 space-y-10">
    <section class="space-y-4">
      <h1 class="text-3xl font-semibold text-gray-900 dark:text-white">
        PowerX Plugin Base 模板介绍
      </h1>
      <p class="text-gray-600 dark:text-gray-300">
        Base 模板插件提供了一个轻量的 CRUD 示范，帮助团队快速理解如何在 PowerX 平台中构建自定义插件。
        您可以使用模板功能为常见业务场景建立可复用的内容片段，加速初始化流程。
      </p>
      <div class="grid gap-4 md:grid-cols-3">
        <UCard>
          <template #header>
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-sparkles" class="text-primary" />
              <span class="font-medium">快速入门</span>
            </div>
          </template>
          <p class="text-sm text-gray-600 dark:text-gray-300">
            通过模板管理示例了解插件的配置、菜单、权限与 Agent 能力声明方式。
          </p>
        </UCard>
        <UCard>
          <template #header>
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-rectangle-group" class="text-primary" />
              <span class="font-medium">可复用模板</span>
            </div>
          </template>
          <p class="text-sm text-gray-600 dark:text-gray-300">
            将常用片段抽象为模板，在多个业务页面中复用，保持体验一致性。
          </p>
        </UCard>
        <UCard>
          <template #header>
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-command-line" class="text-primary" />
              <span class="font-medium">API 演示</span>
            </div>
          </template>
          <p class="text-sm text-gray-600 dark:text-gray-300">
            后端提供最小实现的模板 API，可直接对接 PowerX 平台或扩展为生产级服务。
          </p>
        </UCard>
      </div>
    </section>

    <section class="space-y-4">
      <h2 class="text-2xl font-semibold text-gray-900 dark:text-white">
        下一步
      </h2>
      <div class="grid gap-6 md:grid-cols-2">
        <UCard class="space-y-2">
          <h3 class="text-lg font-medium">探索模板 CRUD</h3>
          <p class="text-sm text-gray-600 dark:text-gray-300">
            在「模版 CRUD」页面体验完整的模板列表、创建、编辑和删除流程。
          </p>
          <UButton to="/templates/crud" color="primary" variant="soft">
            前往模板管理
          </UButton>
        </UCard>
        <UCard class="space-y-2">
          <h3 class="text-lg font-medium">了解开发指引</h3>
          <p class="text-sm text-gray-600 dark:text-gray-300">
            「开发介绍」页面详细说明目录结构、依赖注入和接口注册等关键步骤。
          </p>
          <UButton to="/templates/develop" color="primary" variant="soft">
            查看开发指南
          </UButton>
        </UCard>
      </div>
    </section>
  </UContainer>
</template>

<script setup lang="ts">
definePageMeta({
  title: "PowerXPluginIntro",
})
</script>
