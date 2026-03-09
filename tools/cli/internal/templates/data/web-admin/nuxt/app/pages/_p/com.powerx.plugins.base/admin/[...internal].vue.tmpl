<template>
  <component
    :is="resolvedComponent"
    v-bind="componentProps"
    v-if="resolvedComponent"
    :key="componentKey"
  />
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, watchEffect } from "vue";
import type { Component } from "vue";
import { createError } from "#imports";
import {
  joinLocaleWithPath,
  normalizePath,
  resolveCanonicalInternalPath,
  stripLocalePrefix,
} from "~/utils/powerx-bridge";

const router = useRouter();
const route = useRoute();

definePageMeta({
  layout: "embedded",
});

const internalSegments = computed(() => {
  const value = route.params.internal;
  if (Array.isArray(value)) {
    return value.filter((segment) => typeof segment === "string");
  }
  if (typeof value === "string" && value.length > 0) {
    return [value];
  }
  return [];
});

const internalPath = computed(() => {
  if (!internalSegments.value.length) {
    return "/";
  }
  return `/${internalSegments.value.join("/")}`;
});

const canonicalInternalPath = computed(() =>
  resolveCanonicalInternalPath(internalPath.value)
);

const localeInfo = computed(() => stripLocalePrefix(route.path));

const candidatePaths = computed(() => {
  const localePrefix = localeInfo.value.localePrefix;
  const canonical = canonicalInternalPath.value;

  const candidates = new Set<string>();
  const localeAware = joinLocaleWithPath(localePrefix, canonical);
  candidates.add(localeAware || "/");

  if (canonical === "/dashboard") {
    candidates.add(joinLocaleWithPath(localePrefix, "/"));
  }

  // 兜底：尝试使用无语言前缀的路径，便于开发环境匹配
  candidates.add(joinLocaleWithPath("", canonical));
  candidates.add("/");

  return Array.from(candidates)
    .map((path) => (path ? normalizePath(path) : "/"))
    .filter((path) => path.length > 0);
});

const targetResolution = computed(() => {
  for (const path of candidatePaths.value) {
    const resolved = router.resolve({
      path,
      query: route.query,
      hash: route.hash,
    });

    const record = resolved.matched.at(-1);
    if (record && record.components?.default) {
      return { record, route: resolved };
    }
  }

  return null;
});

watchEffect(() => {
  if (!targetResolution.value) {
    throw createError({ statusCode: 404, statusMessage: "Page not found" });
  }
});

const resolvedComponent = computed<Component | null>(() => {
  const record = targetResolution.value?.record;
  if (!record) {
    return null;
  }

  const component = record.components?.default;
  if (!component) {
    return null;
  }

  if (typeof component === "function" && !("setup" in component)) {
    return defineAsyncComponent(component as () => Promise<Component>);
  }

  return component as Component;
});

const componentProps = computed(() => {
  const resolution = targetResolution.value;
  if (!resolution) {
    return undefined;
  }

  const propsOption = resolution.record.props?.default;
  if (!propsOption) {
    return undefined;
  }

  if (propsOption === true) {
    return resolution.route.params;
  }

  if (typeof propsOption === "function") {
    return propsOption(resolution.route);
  }

  return propsOption;
});

const componentKey = computed(() => {
  const resolution = targetResolution.value;
  if (!resolution) {
    return `embedded-missing:${canonicalInternalPath.value}`;
  }
  const name = resolution.record.name ?? resolution.route.path;
  return `${name}:${canonicalInternalPath.value}`;
});
</script>
