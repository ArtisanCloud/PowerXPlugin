# PowerXPlugin 全局加载适配说明

> 目的：在插件端统一调用接口，不关心运行模式；宿主模式调用 PowerX 底座全局加载，standalone 使用插件本地加载。

## 1. 适配器入口

- 适配器：`web-admin/app/composables/useGlobalLoadingAdapter.ts`

统一接口：

```ts
const gl = useGlobalLoadingAdapter();

gl.show({
  message: "同步中…",
  progress: 0,
  lock: true,
});

gl.setProgress(40);

gl.setMessage("拉取成员详情");

gl.hide();
```

## 2. 运行模式识别

- 通过 `runtimeConfig.public.insidePowerX` 判断是否宿主模式
- 宿主模式：尝试使用 `window.parent.__PX_GLOBAL_LOADING__`
- 如果宿主对象不存在：自动回退本地实现

## 3. 本地实现（standalone）

- 本地全局加载组件：`web-admin/app/components/GlobalLoadingModal.vue`
  - 视觉对齐 PowerX 底座
  - 支持 progress 与无进度两种模式
- 本地 overlay 挂载：`web-admin/app/plugins/gl-overlay.client.ts`
  - 监听本地 loading 状态显示/关闭

## 4. 宿主侧依赖

要在宿主模式生效，底座必须暴露对象：

```ts
window.__PX_GLOBAL_LOADING__ = {
  show,
  hide,
  lock,
  unlock,
  setMessage,
  setProgress,
};
```

否则插件会自动回退至本地 loading。

## 5. 推荐调用规范

### 5.1 无进度加载

```ts
const gl = useGlobalLoadingAdapter();

gl.show({ message: "加载中…", lock: true, minMs: 500 });

gl.hide();
```

### 5.2 有进度加载

```ts
const gl = useGlobalLoadingAdapter();

gl.show({ message: "同步中…", lock: true, progress: 0 });

gl.setProgress(30);

gl.setMessage("写入本地数据");

gl.setProgress(100);

gl.hide();
```

## 6. 典型场景：组织同步进度

- 同步后端持续写入 `sync-logs.progress_percent`
- 前端轮询 `sync-logs`，调用 `gl.setProgress(...)`
- 同步成功/失败时自动 `gl.hide()`

---

如需统一样式升级，请同步修改本地 `GlobalLoadingModal.vue` 与底座版本。
