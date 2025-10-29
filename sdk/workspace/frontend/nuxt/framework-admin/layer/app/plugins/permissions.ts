export default defineNuxtPlugin(() => {
  // 占位权限指令注册点。
  return {
    provide: {
      hasPermission: () => true
    }
  }
})
