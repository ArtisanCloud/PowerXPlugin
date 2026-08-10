export {
  initPowerXBridge,
  PowerXBridgeClient,
  requestPowerXHostAuthToken,
} from "@artisan-cloud/plugin-framework-client";
export type {
  PowerXAuthTokenPayload,
  PowerXBridgeOptions as BridgeOptions,
  PowerXHostMessage as PowerXMessage,
  PowerXPluginToHost as PluginToHost,
  PowerXSyncPayload,
  PowerXThemeKey as ThemeKey,
} from "@artisan-cloud/plugin-framework-client";

export type PowerXFullscreenAction = "enter" | "exit" | "toggle";

export interface PowerXFullscreenRequestOptions {
  pluginId?: string;
  instanceId?: string;
  route?: string;
  reason?: string;
}

export interface PowerXPluginFullscreenPayload {
  source: "powerx-plugin";
  type: "fullscreen:request";
  action: PowerXFullscreenAction;
  pluginId?: string;
  instanceId?: string;
  route?: string;
  reason?: string;
}

export function requestPowerXHostFullscreen(
  pluginId: string,
  options: PowerXFullscreenRequestOptions = {},
) {
  return postPowerXHostFullscreen("enter", pluginId, options);
}

export function exitPowerXHostFullscreen(
  pluginId: string,
  options: PowerXFullscreenRequestOptions = {},
) {
  return postPowerXHostFullscreen("exit", pluginId, options);
}

export function togglePowerXHostFullscreen(
  pluginId: string,
  options: PowerXFullscreenRequestOptions = {},
) {
  return postPowerXHostFullscreen("toggle", pluginId, options);
}

function postPowerXHostFullscreen(
  action: PowerXFullscreenAction,
  pluginId: string,
  options: PowerXFullscreenRequestOptions = {},
) {
  if (typeof window === "undefined" || !window.parent || window.parent === window) {
    return false;
  }
  const payload: PowerXPluginFullscreenPayload = {
    source: "powerx-plugin",
    type: "fullscreen:request",
    action,
    pluginId: options.pluginId || pluginId,
    instanceId: options.instanceId,
    route: options.route,
    reason: options.reason,
  };
  window.parent.postMessage(payload, "*");
  return true;
}
