#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import { createRequire } from "node:module";
import { execFileSync } from "node:child_process";

const PERMISSION_CODE_PATTERN = /^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+:[a-z][a-z0-9_]*$/;
const RISK_LEVELS = new Set(["low", "medium", "high", "critical"]);
const DATA_SCOPES = new Set(["tenant", "global", "system"]);
const API_METHODS = new Set(["GET", "POST", "PUT", "PATCH", "DELETE"]);

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

if (!Array.isArray(permissions) || permissions.length === 0) {
  errors.push(`${permissionSource.label} must declare a non-empty permissions[] array`);
}

const pageBindingPaths = new Set();
const apiBindingKeys = new Set();
const apiBindingEffectiveCodes = new Map();
const actionPermissionCodes = collectPermissionCodesByType(permissions, "action");

for (const [index, permission] of Array.isArray(permissions) ? permissions.entries() : []) {
  const label = `permissions[${index}]`;

  if (!permission || typeof permission !== "object" || Array.isArray(permission)) {
    errors.push(`${label} must be an object permission declaration`);
    continue;
  }

  const type = String(permission.type ?? "");
  if (!["menu", "page", "action", "api"].includes(type)) {
    errors.push(`${label}.type must be one of menu/page/action/api`);
  }

  validatePermissionBasics(permission, label);

  if (type === "page") {
    validatePagePermission(permission, label, pageBindingPaths, pluginId);
  }

  if (type === "api") {
    validateApiPermission(permission, label, pluginId, apiBindingKeys, apiBindingEffectiveCodes, actionPermissionCodes);
  }
}

validateMenuCoverage(manifest, pageBindingPaths);
validateRouteApiCoverage(permissionSource.catalog, apiBindingKeys);

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

  if (!isNonEmptyString(permission.module)) {
    errors.push(`${label}.module must be a non-empty module name`);
  }

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

function validateApiPermission(permission, label, pluginId, apiBindingKeys, apiBindingEffectiveCodes, actionPermissionCodes) {
  const bindings = permission.protocol_bindings;
  if (!Array.isArray(bindings) || bindings.length === 0) {
    errors.push(`${label} type=api must declare non-empty protocol_bindings[]`);
    return;
  }

  const effectivePermissionCode = resolveEffectivePermissionCode(permission, label, actionPermissionCodes);

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

function resolveEffectivePermissionCode(permission, label, actionPermissionCodes) {
  const businessCode = String(permission.business_permission_code ?? "").trim();
  if (businessCode !== "") {
    if (!PERMISSION_CODE_PATTERN.test(businessCode)) {
      errors.push(`${label}.business_permission_code must match ${PERMISSION_CODE_PATTERN}`);
      return "";
    }
    if (!actionPermissionCodes.has(businessCode)) {
      errors.push(`${label}.business_permission_code must reference a declared type=action permission_code: ${businessCode}`);
      return "";
    }
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
}

function validateMenuCoverage(manifest, pageBindingPaths) {
  const menuPaths = collectMenuPaths(manifest?.frontend?.admin?.menus ?? []);
  for (const menuPath of menuPaths) {
    if (!pageBindingPaths.has(menuPath)) {
      errors.push(`frontend.admin.menus path is not covered by a type=page permission binding: ${menuPath}`);
    }
  }
}

function validateRouteApiCoverage(catalog, apiBindingKeys) {
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
    if (!apiBindingKeys.has(routeBindingKey(method, routePath))) {
      errors.push(`${label} ${method} ${routePath} must be covered by a type=api protocol_binding; routes.permissions does not register PowerX Gateway permissions`);
    }
  }
}

function collectMenuPaths(menus) {
  const result = new Set();
  const visit = (items) => {
    if (!Array.isArray(items)) return;
    for (const item of items) {
      if (!item || typeof item !== "object") continue;
      const menuPath = normalizePath(item.path);
      if (menuPath && shouldRequirePageBinding(menuPath)) {
        result.add(menuPath);
      }
      visit(item.children);
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

function routeBindingKey(method, routePath) {
  return `${String(method).toUpperCase()} ${normalizePath(routePath)}`;
}

function isNonEmptyI18n(value) {
  if (!value || typeof value !== "object" || Array.isArray(value)) return false;
  return Object.values(value).some((entry) => typeof entry === "string" && entry.trim().length > 0);
}

function isNonEmptyString(value) {
  return typeof value === "string" && value.trim().length > 0;
}

function isStringArray(value) {
  return Array.isArray(value) && value.every((entry) => isNonEmptyString(entry));
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
