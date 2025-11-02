<template>
  <UContainer class="prose dark:prose-invert max-w-3xl py-10">
    <h1>{{ $t("templates.develop.title") }}</h1>
    <p>{{ $t("templates.develop.intro") }}</p>

    <h2>目录结构</h2>
    <pre><code>app/
  pages/
    intro.vue
    templates/
      index.vue
      crud.vue
      develop.vue
  components/
    templates/
      TemplateFormModal.vue
    ConfirmDialog.vue
    ToastAlert.vue
  composables/
    api/
      useTemplate.ts
</code></pre>

    <h2>数据流</h2>
    <ul>
      <li>组件通过 <code>useTemplateApi</code> 调用后端 API。</li>
      <li>
        <code>ConfirmDialog</code> 与 <code>ToastAlert</code> 用于统一交互反馈。
      </li>
      <li>Tenant ID 默认通过框架中间件注入，前端无需额外处理。</li>
    </ul>

    <h2>后续扩展建议</h2>
    <ul>
      <li>接入权限控制：结合 framework-admin 权限指令。</li>
      <li>接入表格筛选/分页：透过 <code>listTemplates</code> 传参。</li>
      <li>编写 E2E 自动化：使用 Playwright 检验 CRUD 流程。</li>
    </ul>
  </UContainer>
</template>

<script setup lang="ts">
definePageMeta({
  title: "TemplatesDevelopGuide",
});
</script>
