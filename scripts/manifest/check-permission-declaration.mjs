#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import { createRequire } from "node:module";
import { execFileSync } from "node:child_process";

const PERMISSION_CODE_PATTERN = /^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+:[a-z][a-z0-9_]*$/;
const RISK_LEVELS = new Set(["low", "medium", "high", "critical"]);
const DATA_SCOPES = new Set(["tenant", "global", "system"]);
const API_METHODS = new Set(["GET", "POST", "PUT", "PATCH", "DELETE"]);
const PERMISSION_TYPES = new Set(["menu", "page", "action", "data", "api"]);
const HOST_DISCOVERY_ROUTE_PREFIXES = ["/plugin/skills"];

const args = process.argv.slice(2);
const manifestArg = readArg("--manifest") ?? "plugin.yaml";
const manifestPath = path.resolve(process.cwd(), manifestArg);
const repoRoot = findRepoRoot(manifestPath);

const errors = [];

if (!fs.existsSync(manifestPath)) {
  fail([`manifest not found: ${manifestArg}`]);
}

const manifest = loadManifest(manifestPath, repoRoot);
const pluginId = String(manifest?.id ?? "");
const permissionSource = loadPermissionSource(manifest, manifestPath, repoRoot);
const permissions = permissionSource.permissions;
const forbiddenPermissionFragments = buildForbiddenPermissionFragments(pluginId);

if (!Array.isArray(permissions) || permissions.length === 0) {
  errors.push(`${permissionSource.label} must declare a non-empty permissions[] array`);
}

const pageBindingPaths = new Set();
const apiBindingKeys = new Set();
const apiBindingEffectiveCodes = new Map();
const pagePermissionCodes = collectPermissionCodesByType(permissions, "page");
const menuPermissionCodes = collectPermissionCodesByType(permissions, "menu");
const operationPermissionCodes = collectPermissionCodesByTypes(permissions, ["page", "action", "data"]);
const menuPermissionPaths = new Map();

validateDuplicatePermissionCodes(permissions);

for (const [index, permission] of Array.isArray(permissions) ? permissions.entries() : []) {
  const label = `permissions[${index}]`;

  if (!permission || typeof permission !== "object" || Array.isArray(permission)) {
    errors.push(`${label} must be an object permission declaration`);
    continue;
  }

  const type = String(permission.type ?? "");
  if (!PERMISSION_TYPES.has(type)) {
    errors.push(`${label}.type must be one of menu/page/action/data/api`);
  }

  validatePermissionBasics(permission, label);

  if (type === "menu") {
    validateMenuPermission(permission, label, pagePermissionCodes, menuPermissionPaths);
  }

  if (type === "page") {
    validateStructuredPermission(permission, label);
    validatePagePermission(permission, label, pageBindingPaths, pluginId);
  }

  if (type === "action" || type === "data") {
    validateStructuredPermission(permission, label);
  }

  if (type === "api") {
    validateStructuredPermission(permission, label);
    validateApiPermission(permission, label, pluginId, apiBindingKeys, apiBindingEffectiveCodes, operationPermissionCodes);
    validateApiDisplayI18n(permission, label);
  }
}

validateMenuCoverage(manifest, pageBindingPaths, menuPermissionCodes, menuPermissionPaths);
validateRouteApiCoverage(permissionSource.catalog, apiBindingKeys, apiBindingEffectiveCodes);
validateLegacyRBACResources(permissionSource.catalog);
validateEventFabricPublishACL(manifest, manifestPath, repoRoot, pluginId);

if (errors.length > 0) {
  fail(errors);
}

console.log(`[manifest] permission declaration check passed: ${permissions.length} permissions, ${pageBindingPaths.size} page bindings (${permissionSource.label})`);

function readArg(name) {
  const index = args.indexOf(name);
  if (index === -1) return null;
  return args[index + 1] ?? null;
}

function loadPermissionSource(manifest, manifestPath, repoRoot) {
  const rbacCatalogPath = String(manifest?.catalogs?.rbac ?? "").trim();
  if (!rbacCatalogPath) {
    return {
      label: "plugin.yaml",
      permissions: manifest?.permissions,
    };
  }

  for (const field of ["permissions", "rbac", "routes"]) {
    if (manifest?.[field] !== undefined && !isEmptyValue(manifest[field])) {
      errors.push(`catalog conflict on field "${field}" (catalog=rbac): move ${field} into ${rbacCatalogPath}`);
    }
  }

  const catalogPath = resolveManifestRelativePath(manifestPath, rbacCatalogPath);
  if (!fs.existsSync(catalogPath)) {
    errors.push(`catalogs.rbac not found: ${rbacCatalogPath}`);
    return {
      label: rbacCatalogPath,
      permissions: undefined,
    };
  }

  const catalog = loadManifest(catalogPath, repoRoot);
  return {
    label: rbacCatalogPath,
    catalog,
    permissions: catalog?.permissions,
  };
}

function validatePermissionBasics(permission, label) {
  const code = String(permission.permission_code ?? "");
  if (!PERMISSION_CODE_PATTERN.test(code)) {
    errors.push(`${label}.permission_code must match ${PERMISSION_CODE_PATTERN}`);
  }
  validateNoPluginIdentity(code, `${label}.permission_code`);

  if (!isNonEmptyString(permission.module)) {
    errors.push(`${label}.module must be a non-empty module name`);
  }
  validateNoPluginIdentity(permission.module, `${label}.module`);

  if (!isNonEmptyI18n(permission.title_i18n)) {
    errors.push(`${label}.title_i18n must contain at least one non-empty locale string`);
  }

  if (!isNonEmptyI18n(permission.description_i18n)) {
    errors.push(`${label}.description_i18n must contain at least one non-empty locale string`);
  }

  if (!RISK_LEVELS.has(String(permission.risk_level ?? ""))) {
    errors.push(`${label}.risk_level must be one of low/medium/high/critical`);
  }

  const dataScope = String(permission.data_scope ?? "");
  if (!DATA_SCOPES.has(dataScope)) {
    errors.push(`${label}.data_scope must be one of ${Array.from(DATA_SCOPES).join("/")}`);
  }

  if (permission.default_role_grants !== undefined && !isStringArray(permission.default_role_grants)) {
    errors.push(`${label}.default_role_grants must be an array of role code strings when declared`);
  }

  if (permission.source !== undefined) {
    errors.push(`${label}.source must not be declared by plugins; PowerX derives source from plugin_id / iam_permission.source`);
  }
}

function validateMenuPermission(permission, label, pagePermissionCodes, menuPermissionPaths) {
  if (!isNonEmptyStringArray(permission.menu_path)) {
    errors.push(`${label} type=menu must declare menu_path as a non-empty string array`);
  } else {
    for (const [index, segment] of permission.menu_path.entries()) {
      validateNoPluginIdentity(segment, `${label}.menu_path[${index}]`);
      validateNoMenuPathReservedSegment(segment, `${label}.menu_path[${index}]`);
      if (!/^[a-z][a-z0-9_]*$/.test(segment)) {
        errors.push(`${label}.menu_path[${index}] must be a stable lowercase business key`);
      }
    }
    const code = String(permission.permission_code ?? "").trim();
    if (code) {
      menuPermissionPaths.set(code, permission.menu_path.map((segment) => String(segment).trim()));
    }
  }

  if (!isNonEmptyStringArray(permission.page_permission_codes)) {
    errors.push(`${label} type=menu must declare page_permission_codes as a non-empty array of page permission_code strings`);
  } else {
    for (const [index, code] of permission.page_permission_codes.entries()) {
      if (!pagePermissionCodes.has(code)) {
        errors.push(`${label}.page_permission_codes[${index}] must reference a declared type=page permission_code: ${code}`);
      }
    }
  }
}

function validateStructuredPermission(permission, label) {
  for (const field of ["resource", "action"]) {
    if (!isNonEmptyString(permission[field])) {
      errors.push(`${label}.${field} must be declared for type=${permission.type}`);
      continue;
    }
    validateNoPluginIdentity(permission[field], `${label}.${field}`);
    if (!/^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$/.test(String(permission[field]))) {
      errors.push(`${label}.${field} must be a lowercase business key`);
    }
  }

  const parsed = splitPermissionCode(permission.permission_code);
  if (!parsed) return;
  const expectedLeft = `${String(permission.module ?? "").trim()}.${String(permission.resource ?? "").trim()}`;
  const expected = `${expectedLeft}:${String(permission.action ?? "").trim()}`;
  if (permission.permission_code !== expected) {
    errors.push(`${label}.permission_code must equal module.resource:action (${expected})`);
  }
}

function validatePagePermission(permission, label, pageBindingPaths, pluginId) {
  const bindings = permission.protocol_bindings;
  if (!Array.isArray(bindings) || bindings.length === 0) {
    errors.push(`${label} type=page must declare non-empty protocol_bindings[]`);
    return;
  }

  for (const [index, binding] of bindings.entries()) {
    const bindingLabel = `${label}.protocol_bindings[${index}]`;
    validateBindingShape(binding, bindingLabel, pluginId);
    if (String(binding?.method ?? "").toUpperCase() !== "GET") {
      errors.push(`${bindingLabel}.method must be GET for type=page`);
    }
    const normalizedPath = normalizePath(binding?.path);
    if (normalizedPath) {
      pageBindingPaths.add(normalizedPath);
    }
  }
}

function validateApiPermission(permission, label, pluginId, apiBindingKeys, apiBindingEffectiveCodes, operationPermissionCodes) {
  const bindings = permission.protocol_bindings;
  if (!Array.isArray(bindings) || bindings.length === 0) {
    errors.push(`${label} type=api must declare non-empty protocol_bindings[]`);
    return;
  }

  const effectivePermissionCode = resolveEffectivePermissionCode(permission, label, operationPermissionCodes);

  for (const [index, binding] of bindings.entries()) {
    const bindingLabel = `${label}.protocol_bindings[${index}]`;
    validateBindingShape(binding, bindingLabel, pluginId);
    const method = String(binding?.method ?? "").toUpperCase();
    if (!API_METHODS.has(method)) {
      errors.push(`${bindingLabel}.method must be one of ${Array.from(API_METHODS).join("/")}`);
    }
    const normalizedPath = normalizePath(binding?.path);
    if (method && normalizedPath) {
      const bindingKey = routeBindingKey(method, normalizedPath);
      apiBindingKeys.add(bindingKey);
      if (effectivePermissionCode) {
        const previousEffectiveCode = apiBindingEffectiveCodes.get(bindingKey);
        if (previousEffectiveCode && previousEffectiveCode !== effectivePermissionCode) {
          errors.push(`${bindingLabel} ${method} ${normalizedPath} is already bound to effective_permission_code ${previousEffectiveCode}; duplicate route bindings must resolve to one effective permission`);
        } else {
          apiBindingEffectiveCodes.set(bindingKey, effectivePermissionCode);
        }
      }
    }
  }
}

function validateApiDisplayI18n(permission, label) {
  const code = String(permission.permission_code ?? "").trim();
  const technicalValues = [
    code,
    `/_p/${pluginId}`,
    "/_p/",
    "/api/v1",
  ].filter(Boolean);

  const bindings = Array.isArray(permission.protocol_bindings) ? permission.protocol_bindings : [];
  for (const binding of bindings) {
    const method = String(binding?.method ?? "").trim().toUpperCase();
    const bindingPath = normalizePath(binding?.path);
    if (bindingPath) {
      technicalValues.push(bindingPath);
      if (method) {
        technicalValues.push(`${method} ${bindingPath}`);
      }
    }
  }

  for (const field of ["title_i18n", "description_i18n"]) {
    for (const [locale, value] of i18nEntries(permission[field])) {
      const normalized = value.trim();
      if (!normalized) continue;
      for (const technicalValue of technicalValues) {
        if (normalized === technicalValue) {
          errors.push(`${label}.${field}.${locale} must be user-readable copy, not technical API metadata: ${technicalValue}`);
        }
      }
    }
  }
}

function resolveEffectivePermissionCode(permission, label, operationPermissionCodes) {
  const businessCode = String(permission.business_permission_code ?? "").trim();
  if (businessCode !== "") {
    if (permission.independent === true) {
      errors.push(`${label} type=api must not declare independent: true when business_permission_code is set`);
    }
    if (!PERMISSION_CODE_PATTERN.test(businessCode)) {
      errors.push(`${label}.business_permission_code must match ${PERMISSION_CODE_PATTERN}`);
      return "";
    }
    if (!operationPermissionCodes.has(businessCode)) {
      errors.push(`${label}.business_permission_code must reference a declared type=page/action/data permission_code: ${businessCode}`);
      return "";
    }
    validateNoPluginIdentity(businessCode, `${label}.business_permission_code`);
    return businessCode;
  }

  if (permission.independent === true) {
    const apiCode = String(permission.permission_code ?? "").trim();
    if (!PERMISSION_CODE_PATTERN.test(apiCode)) {
      errors.push(`${label}.permission_code must be valid when used as independent effective_permission_code`);
      return "";
    }
    return apiCode;
  }

  errors.push(`${label} type=api must resolve effective_permission_code by declaring business_permission_code or independent: true`);
  return "";
}

function validateBindingShape(binding, label, pluginId) {
  if (!binding || typeof binding !== "object" || Array.isArray(binding)) {
    errors.push(`${label} must be an object`);
    return;
  }

  if (String(binding.channel ?? "") !== "rest") {
    errors.push(`${label}.channel must be rest`);
  }

  const method = String(binding.method ?? "").toUpperCase();
  if (!method) {
    errors.push(`${label}.method is required`);
  }

  if (String(binding.actor_context ?? "") !== "admin_user") {
    errors.push(`${label}.actor_context must be admin_user`);
  }

  if (String(binding.resource_scope ?? "") !== "tenant") {
    errors.push(`${label}.resource_scope must be tenant`);
  }

  validateBindingPath(binding.path, label, pluginId);
}

function validateBindingPath(rawPath, label, pluginId) {
  const bindingPath = normalizePath(rawPath);
  if (!bindingPath) {
    errors.push(`${label}.path must be an absolute path`);
    return;
  }

  const forbiddenPrefixes = [
    `/_p/${pluginId}`,
    "/_p/",
    "/api/v1",
    "/api/",
    "/api",
    "/v1",
  ].filter(Boolean);

  for (const prefix of forbiddenPrefixes) {
    if (pathHasPrefix(bindingPath, prefix)) {
      errors.push(`${label}.path must not include host/plugin prefix '${prefix}': ${bindingPath}`);
      return;
    }
  }

  if (/[{][^/{}]+[}]/.test(bindingPath) || /(^|\/):[^/]+/.test(bindingPath)) {
    errors.push(`${label}.path must use '*' for dynamic segments, not '{param}' or ':param': ${bindingPath}`);
  }

  if (isHostDiscoveryRoute(bindingPath)) {
    errors.push(`${label}.path must not declare host discovery route ${bindingPath}; PowerX calls these during install before user authorization snapshots exist`);
  }
}

function validateMenuCoverage(manifest, pageBindingPaths, menuPermissionCodes, menuPermissionPaths) {
  const menuItems = collectMenuItems(manifest?.frontend?.admin?.menus ?? []);
  const referencedMenuPermissionCodes = new Set();

  for (const item of menuItems) {
    if (item.pagePath && !pageBindingPaths.has(item.pagePath)) {
      errors.push(`frontend.admin.menus path is not covered by a type=page permission binding: ${item.pagePath}`);
    }

    for (const [index, policy] of item.requiredPolicies.entries()) {
      if (!menuPermissionCodes.has(policy)) {
        errors.push(`frontend.admin.menus required_policies[${index}] references no declared type=menu permission: ${policy} (menu_path=${item.idPath.join("/") || "<root>"})`);
        continue;
      }
      referencedMenuPermissionCodes.add(policy);
      const expectedPath = item.idPath;
      const actualPath = menuPermissionPaths.get(policy) ?? [];
      if (!sameStringArray(actualPath, expectedPath)) {
        errors.push(`type=menu permission ${policy} menu_path mismatch: expected ${expectedPath.join("/")}, got ${actualPath.join("/") || "<empty>"}`);
      }
    }
  }

  for (const code of menuPermissionCodes) {
    if (!referencedMenuPermissionCodes.has(code)) {
      errors.push(`type=menu permission is not referenced by any frontend.admin.menus required_policies: ${code}`);
    }
  }
}

function validateRouteApiCoverage(catalog, apiBindingKeys, apiBindingEffectiveCodes) {
  const routePermissions = catalog?.routes?.permissions;
  if (!Array.isArray(routePermissions) || routePermissions.length === 0) return;

  for (const [index, route] of routePermissions.entries()) {
    const label = `routes.permissions[${index}]`;
    if (!route || typeof route !== "object" || Array.isArray(route)) {
      errors.push(`${label} must be an object`);
      continue;
    }

    const method = String(route.method ?? "").toUpperCase();
    const routePath = normalizePath(route.path);
    if (!API_METHODS.has(method)) {
      errors.push(`${label}.method must be one of ${Array.from(API_METHODS).join("/")}`);
      continue;
    }
    if (!routePath) {
      errors.push(`${label}.path must be an absolute path`);
      continue;
    }
    validateBindingPath(route.path, label, pluginId);
    const key = routeBindingKey(method, routePath);
    if (!apiBindingKeys.has(key)) {
      errors.push(`${label} ${method} ${routePath} must be covered by a type=api protocol_binding; routes.permissions does not register PowerX Gateway permissions`);
      continue;
    }
    const routePermission = permissionFromRouteEntry(route);
    const effectiveCode = apiBindingEffectiveCodes.get(key);
    if (routePermission && effectiveCode && routePermission !== effectiveCode) {
      errors.push(`${label} ${method} ${routePath} must map to effective_permission_code ${effectiveCode}, got ${routePermission}`);
    }
  }
}

function validateLegacyRBACResources(catalog) {
  const resources = catalog?.rbac?.resources;
  if (!Array.isArray(resources)) return;
  for (const [index, entry] of resources.entries()) {
    if (!entry || typeof entry !== "object" || Array.isArray(entry)) continue;
    validateNoPluginIdentity(entry.resource, `rbac.resources[${index}].resource`);
  }
}

function validateEventFabricPublishACL(manifest, manifestPath, repoRoot, pluginId) {
  const eventsCatalogPath = String(manifest?.catalogs?.events ?? "").trim();
  if (!eventsCatalogPath) return;

  const eventsPath = resolveManifestRelativePath(manifestPath, eventsCatalogPath);
  if (!fs.existsSync(eventsPath)) return;

  const eventsCatalog = loadManifest(eventsPath, repoRoot);
  const publishTopics = new Set();
  const topics = eventsCatalog?.events?.topics;
  if (!Array.isArray(topics)) return;

  for (const topic of topics) {
    if (!topic || typeof topic !== "object" || Array.isArray(topic)) continue;
    const key = String(topic.key ?? "").trim();
    const actions = Array.isArray(topic.actions) ? topic.actions.map((action) => String(action).trim().toLowerCase()) : [];
    if (key && actions.includes("publish")) {
      publishTopics.add(key);
    }
  }
  if (publishTopics.size === 0) return;

  const fabricPath = path.resolve(path.dirname(manifestPath), "config/event_fabric.yaml");
  if (!fs.existsSync(fabricPath)) {
    errors.push(`config/event_fabric.yaml is required because ${eventsCatalogPath} declares publish topics`);
    return;
  }

  const fabric = loadManifest(fabricPath, repoRoot);
  const fabricTopics = new Map();
  for (const topic of Array.isArray(fabric?.topics) ? fabric.topics : []) {
    if (!topic || typeof topic !== "object" || Array.isArray(topic)) continue;
    const key = String(topic.key ?? "").trim();
    if (key) fabricTopics.set(key, topic);
  }

  const pluginPrincipal = `plugin:${pluginId}`;
  for (const topicKey of publishTopics) {
    const topic = fabricTopics.get(topicKey);
    if (!topic) {
      errors.push(`event_fabric topic missing for publish topic: ${topicKey}`);
      continue;
    }
    const acl = Array.isArray(topic.acl) ? topic.acl : [];
    const hasPluginPublish = acl.some((entry) => {
      if (!entry || typeof entry !== "object" || Array.isArray(entry)) return false;
      const principalType = String(entry.principal_type ?? "").trim();
      const principalID = String(entry.principal_id ?? "").trim();
      const actions = Array.isArray(entry.actions) ? entry.actions.map((action) => String(action).trim().toLowerCase()) : [];
      return principalType === "plugin" && principalID === pluginPrincipal && actions.includes("publish");
    });
    if (!hasPluginPublish) {
      errors.push(`event_fabric topic ${topicKey} must grant plugin principal ${pluginPrincipal} publish`);
    }
  }
}

function collectMenuItems(menus) {
  const result = [];
  const visit = (items, ancestors = []) => {
    if (!Array.isArray(items)) return;
    for (const item of items) {
      if (!item || typeof item !== "object") continue;
      const id = String(item.id ?? "").trim();
      if (id) {
        validateMenuItemID(id, "frontend.admin.menus[].id");
      }
      const idPath = id ? [...ancestors, id] : [...ancestors];
      const menuPath = normalizePath(item.path);
      const requiredPolicies = Array.isArray(item.required_policies)
        ? item.required_policies.map((policy) => String(policy).trim()).filter(Boolean)
        : [];
      if (menuPath && shouldRequirePageBinding(menuPath)) {
        result.push({ idPath, pagePath: menuPath, requiredPolicies });
      } else if (requiredPolicies.length > 0) {
        result.push({ idPath, pagePath: "", requiredPolicies });
      }
      visit(item.children, idPath);
    }
  };
  visit(menus);
  return result;
}

function shouldRequirePageBinding(menuPath) {
  if (/^https?:\/\//.test(menuPath)) return false;
  return ![
    "/healthz",
    "/health",
    "/bridge-dev",
    "/assets",
    "/_nuxt",
  ].some((prefix) => menuPath === prefix || menuPath.startsWith(`${prefix}/`));
}

function normalizePath(value) {
  if (typeof value !== "string") return "";
  const trimmed = value.trim();
  if (!trimmed || !trimmed.startsWith("/")) return "";
  if (trimmed.length > 1 && trimmed.endsWith("/")) {
    return trimmed.replace(/\/+$/, "");
  }
  return trimmed;
}

function resolveManifestRelativePath(manifestPath, rawPath) {
  if (path.isAbsolute(rawPath)) return rawPath;
  return path.resolve(path.dirname(manifestPath), rawPath);
}

function isEmptyValue(value) {
  if (value === undefined || value === null) return true;
  if (Array.isArray(value)) return value.length === 0;
  if (typeof value === "object") return Object.keys(value).length === 0;
  if (typeof value === "string") return value.trim() === "";
  return false;
}

function pathHasPrefix(targetPath, prefix) {
  const normalizedPrefix = prefix.endsWith("/") && prefix !== "/" ? prefix.slice(0, -1) : prefix;
  return targetPath === normalizedPrefix || targetPath.startsWith(`${normalizedPrefix}/`);
}

function isHostDiscoveryRoute(targetPath) {
  return HOST_DISCOVERY_ROUTE_PREFIXES.some((prefix) => pathHasPrefix(targetPath, prefix));
}

function routeBindingKey(method, routePath) {
  return `${String(method).toUpperCase()} ${normalizePath(routePath)}`;
}

function permissionFromRouteEntry(route) {
  const resource = String(route?.resource ?? "").trim();
  const action = String(route?.action ?? "").trim();
  if (!resource || !action) return "";
  validateNoPluginIdentity(resource, "routes.permissions[].resource");
  validateNoPluginIdentity(action, "routes.permissions[].action");
  return `${resource}:${action}`;
}

function isNonEmptyI18n(value) {
  if (!value || typeof value !== "object" || Array.isArray(value)) return false;
  return Object.values(value).some((entry) => typeof entry === "string" && entry.trim().length > 0);
}

function i18nEntries(value) {
  if (!value || typeof value !== "object" || Array.isArray(value)) return [];
  return Object.entries(value).filter(([, entry]) => typeof entry === "string");
}

function isNonEmptyString(value) {
  return typeof value === "string" && value.trim().length > 0;
}

function isStringArray(value) {
  return Array.isArray(value) && value.every((entry) => isNonEmptyString(entry));
}

function isNonEmptyStringArray(value) {
  return isStringArray(value) && value.length > 0;
}

function validateDuplicatePermissionCodes(permissions) {
  const seen = new Set();
  for (const [index, permission] of Array.isArray(permissions) ? permissions.entries() : []) {
    const code = String(permission?.permission_code ?? "").trim();
    if (!code) continue;
    if (seen.has(code)) {
      errors.push(`permissions[${index}].permission_code duplicates another declaration: ${code}`);
    }
    seen.add(code);
  }
}

function collectPermissionCodesByType(permissions, type) {
  const codes = new Set();
  if (!Array.isArray(permissions)) return codes;
  for (const permission of permissions) {
    if (!permission || typeof permission !== "object" || Array.isArray(permission)) continue;
    if (String(permission.type ?? "") !== type) continue;
    const code = String(permission.permission_code ?? "").trim();
    if (code) {
      codes.add(code);
    }
  }
  return codes;
}

function collectPermissionCodesByTypes(permissions, types) {
  const wanted = new Set(types);
  const codes = new Set();
  if (!Array.isArray(permissions)) return codes;
  for (const permission of permissions) {
    if (!permission || typeof permission !== "object" || Array.isArray(permission)) continue;
    if (!wanted.has(String(permission.type ?? ""))) continue;
    const code = String(permission.permission_code ?? "").trim();
    if (code) {
      codes.add(code);
    }
  }
  return codes;
}

function splitPermissionCode(code) {
  const raw = String(code ?? "").trim();
  const match = raw.match(/^(.+):([^:]+)$/);
  if (!match) return null;
  return { left: match[1], action: match[2] };
}

function buildForbiddenPermissionFragments(pluginId) {
  const fragments = new Set();
  const raw = String(pluginId ?? "").trim();
  if (raw) {
    fragments.add(raw.toLowerCase());
    fragments.add(raw.replace(/[-.]/g, "_").toLowerCase());
  }
  fragments.add("com.powerx.plugins.base");
  fragments.add("com_powerx_plugins_base");
  return Array.from(fragments).filter(Boolean);
}

function validateNoPluginIdentity(value, label) {
  if (value === undefined || value === null) return;
  const raw = String(value).trim().toLowerCase();
  if (!raw) return;
  for (const fragment of forbiddenPermissionFragments) {
    if (raw.includes(fragment)) {
      errors.push(`${label} must not include plugin id or plugin-derived namespace: ${value}`);
      return;
    }
  }
}

function validateNoMenuPathReservedSegment(value, label) {
  const raw = String(value ?? "").trim().toLowerCase();
  if (!raw) return;
  if (raw === "apps" || raw === "_p" || raw === "api" || raw === "api_v1" || raw === "v1") {
    errors.push(`${label} must not include APPS, _p, API path, or host route segments: ${value}`);
  }
  if (raw.includes("/") || raw.includes(":")) {
    errors.push(`${label} must be a menu key, not a URL or API path segment: ${value}`);
  }
}

function validateMenuItemID(value, label) {
  validateNoPluginIdentity(value, label);
  validateNoMenuPathReservedSegment(value, label);
  if (!/^[a-z][a-z0-9_]*$/.test(String(value ?? ""))) {
    errors.push(`${label} must be a stable lowercase business key: ${value}`);
  }
}

function sameStringArray(left, right) {
  if (!Array.isArray(left) || !Array.isArray(right) || left.length !== right.length) return false;
  return left.every((value, index) => value === right[index]);
}

function loadManifest(filePath, rootDir) {
  const yaml = loadYaml(rootDir);
  if (yaml) {
    return yaml.parse(fs.readFileSync(filePath, "utf8"));
  }

  try {
    const output = execFileSync(
      "ruby",
      [
        "-ryaml",
        "-rjson",
        "-e",
        "puts JSON.generate(YAML.load_file(ARGV.fetch(0)))",
        filePath,
      ],
      { encoding: "utf8" },
    );
    return JSON.parse(output);
  } catch {
    fail([
      "cannot load npm package 'yaml' and Ruby YAML fallback is unavailable",
      "run npm install at the repo root, install frontend dependencies, or install Ruby",
    ]);
  }
}

function loadYaml(rootDir) {
  const requireFromHere = createRequire(import.meta.url);
  const candidates = [
    "yaml",
    path.join(rootDir, "node_modules/yaml"),
    path.join(rootDir, "web-admin/node_modules/yaml"),
  ];

  for (const candidate of candidates) {
    try {
      return requireFromHere(candidate);
    } catch {
      // Try the next candidate.
    }
  }

  return null;
}

function findRepoRoot(filePath) {
  let current = path.dirname(filePath);
  while (current !== path.dirname(current)) {
    if (fs.existsSync(path.join(current, "plugin.yaml"))) {
      return current;
    }
    current = path.dirname(current);
  }
  return process.cwd();
}

function fail(messages) {
  console.error("[manifest] permission declaration check failed");
  for (const message of messages) {
    console.error(` - ${message}`);
  }
  process.exit(1);
}
