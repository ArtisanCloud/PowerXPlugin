<script setup lang="ts">
definePageMeta({ layout: 'embedded', title: 'Host Simulator' })

const { t } = useI18n({ useScope: 'local' })
const locale = ref('zh')
const theme  = ref<'light'|'dark'|'system'>('light')
const iframeRef = ref<HTMLIFrameElement | null>(null)
const log = ref<string>('')
const fullscreen = ref(false)
const fullscreenAction = ref<'enter' | 'exit' | 'toggle' | ''>('')

const append = (...args: any[]) => {
  const line = args.map(a => (typeof a === 'string' ? a : JSON.stringify(a))).join(' ')
  log.value += line + '\n'
}

const post = (msg: any) => {
  const win = iframeRef.value?.contentWindow
  if (!win) return
  // 本地演示用 '*'; 真实宿主中应使用 new URL(iframe.src).origin
  win.postMessage(msg, '*')
  append('[host] =>', msg)
}

const sendLocale = () => post({ source: 'powerx', type: 'locale', locale: locale.value })
const sendTheme  = () => post({ source: 'powerx', type: 'theme',  theme: theme.value  })
const sendSync   = () => post({
  source: 'powerx', type: 'sync',
  locale: locale.value, theme: theme.value,
  hostOrigin: window.location.origin,
  pluginId: 'com.powerx.plugins.base', instanceId: 'dev-bridge'
})

const applyFullscreenRequest = (data: any) => {
  const action = data?.action as 'enter' | 'exit' | 'toggle'
  fullscreenAction.value = action || ''
  if (action === 'enter') {
    fullscreen.value = true
  } else if (action === 'exit') {
    fullscreen.value = false
  } else if (action === 'toggle') {
    fullscreen.value = !fullscreen.value
  }
  append('[host] fullscreen =>', {
    action,
    active: fullscreen.value,
    pluginId: data?.pluginId,
    instanceId: data?.instanceId,
    route: data?.route,
    reason: data?.reason
  })
}

onMounted(() => {
  window.addEventListener('message', (e: MessageEvent) => {
    const data = (e.data || {}) as any
    if (data?.source === 'plugin') {
      append('[host] <=', data)
      if (data.type === 'request-sync') sendSync()
      return
    }
    if (data?.source === 'powerx-plugin') {
      append('[host] <=', data)
      if (data.type === 'fullscreen:request') applyFullscreenRequest(data)
    }
  }, false)
})
</script>

<template>
  <div class="wrap" :class="{ 'is-fullscreen': fullscreen }">
    <h2>{{ t('host.title') }}</h2>
    <div class="row">
      <label>{{ t('host.locale') }} <input v-model="locale" /></label>
      <label>{{ t('host.theme') }}
        <select v-model="theme">
          <option value="light">light</option>
          <option value="dark">dark</option>
          <option value="system">system</option>
        </select>
      </label>
      <button @click="sendLocale">{{ t('host.broadcastLocale') }}</button>
      <button @click="sendTheme">{{ t('host.broadcastTheme') }}</button>
      <button @click="sendSync">{{ t('host.broadcastSync') }}</button>
      <span class="status" :data-active="fullscreen">
        {{ fullscreen ? t('host.fullscreenOn') : t('host.fullscreenOff') }}
        <small v-if="fullscreenAction">({{ fullscreenAction }})</small>
      </span>
    </div>

    <iframe
      ref="iframeRef"
      src="/bridge-dev/plugin"
      referrerpolicy="strict-origin-when-cross-origin"
      class="demo-iframe"
      :class="{ 'demo-iframe-fullscreen': fullscreen }"
    />

    <pre class="log">{{ log }}</pre>
  </div>
</template>

<i18n lang="json">
{
  "zh": {
    "host": {
      "title": "Host Simulator (PowerX mock)",
      "locale": "Locale:",
      "theme": "Theme:",
      "broadcastLocale": "Broadcast locale",
      "broadcastTheme": "Broadcast theme",
      "broadcastSync": "Broadcast sync",
      "fullscreenOn": "Fullscreen active",
      "fullscreenOff": "Fullscreen inactive"
    }
  },
  "en": {
    "host": {
      "title": "Host Simulator (PowerX mock)",
      "locale": "Locale:",
      "theme": "Theme:",
      "broadcastLocale": "Broadcast locale",
      "broadcastTheme": "Broadcast theme",
      "broadcastSync": "Broadcast sync",
      "fullscreenOn": "Fullscreen active",
      "fullscreenOff": "Fullscreen inactive"
    }
  }
}
</i18n>

<style scoped>
.wrap { padding:16px; font-family: ui-monospace, Menlo, monospace; }
.row { display:flex; gap:12px; align-items:center; flex-wrap:wrap; margin-bottom:12px; }
button { padding:6px 10px; border-radius:8px; border:1px solid #ccc; background:#f0f0f0; cursor:pointer; }
.demo-iframe { width:100%; height:420px; border:1px solid #ddd; }
.demo-iframe-fullscreen { height: calc(100vh - 180px); border-color:#16a34a; box-shadow:0 0 0 2px rgba(22,163,74,.25); }
.status { padding:6px 10px; border-radius:8px; border:1px solid #ccc; background:#f8fafc; color:#334155; }
.status[data-active="true"] { border-color:#16a34a; background:#dcfce7; color:#166534; }
.log { background:#111; color:#0f0; padding:8px; margin-top:12px; height:180px; overflow:auto; white-space:pre-wrap; }
</style>
