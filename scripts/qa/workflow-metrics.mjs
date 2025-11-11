#!/usr/bin/env node
/**
 * workflow-metrics.mjs
 * 轻量级脚本：读取 Prometheus/Grafana 导出的 JSON 或本地样例，输出 Publish Hub 关注的关键指标。
 * 用法：
 *   node scripts/qa/workflow-metrics.mjs --input ./reports/metrics.json --scope publish
 */

import { readFile } from "node:fs/promises";
import path from "node:path";

const args = process.argv.slice(2);
const options = {};
for (let i = 0; i < args.length; i += 1) {
  const arg = args[i];
  if (arg.startsWith("--")) {
    const key = arg.replace(/^--/, "");
    options[key] = args[i + 1];
    i += 1;
  }
}

async function loadMetrics(file) {
  if (!file) {
    return {
      dev_hotload_reload_duration_ms: 1800,
      plugin_publish_pipeline_duration_ms: 3600000,
      publish_local_iteration_cycle_time: 900000,
      publish_gray_error_rate: 0.01,
      marketplace_listing_sla_hours: 12,
    };
  }
  const resolved = path.resolve(file);
  const data = await readFile(resolved, "utf-8");
  return JSON.parse(data);
}

function formatMetrics(metrics) {
  return [
    {
      key: "publish_local_iteration_cycle_time",
      label: "Local iteration cycle",
      target: "<= 15m",
      value: msToMinutes(metrics.publish_local_iteration_cycle_time),
    },
    {
      key: "publish_gray_error_rate",
      label: "Gray deploy error rate",
      target: "< 5%",
      value: `${(metrics.publish_gray_error_rate * 100).toFixed(2)}%`,
    },
    {
      key: "marketplace_listing_sla_hours",
      label: "Marketplace listing SLA",
      target: "<= 72h",
      value: `${metrics.marketplace_listing_sla_hours ?? "n/a"}h`,
    },
    {
      key: "plugin_publish_pipeline_duration_ms",
      label: "Publish pipeline duration",
      target: "<= 4h",
      value: `${msToHours(metrics.plugin_publish_pipeline_duration_ms)}h`,
    },
  ];
}

function msToMinutes(value) {
  if (!value && value !== 0) return "n/a";
  return `${(value / 60000).toFixed(2)}m`;
}

function msToHours(value) {
  if (!value && value !== 0) return "n/a";
  return (value / (1000 * 60 * 60)).toFixed(2);
}

const scope = options.scope ?? "publish";
const input = options.input;

const metrics = await loadMetrics(input);
const table = formatMetrics(metrics);

console.log(`Workflow Metrics (${scope})`);
console.log("-------------------------------------------");
for (const row of table) {
  console.log(`${row.label.padEnd(32)} value=${row.value} (target ${row.target})`);
}
