<template>
  <div class="px-admin-layout">
    <header class="px-admin-layout__header">
      <slot name="toolbar">
        <h1>PowerX Admin</h1>
      </slot>
    </header>
    <main class="px-admin-layout__body">
      <slot />
    </main>
  </div>
</template>

<style scoped>
.px-admin-layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.px-admin-layout__header {
  padding: 1rem 2rem;
  border-bottom: 1px solid #eee;
}

.px-admin-layout__body {
  flex: 1;
  padding: 2rem;
}
</style>
