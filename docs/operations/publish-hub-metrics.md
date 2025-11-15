# Publish Hub Metrics & SLA Dashboard

## Overview

This document describes the Prometheus metrics exported by the Publish Hub system for monitoring SLAs and deployment health.

## Metrics

### SC-002: Online Publish Review Duration
- **Metric**: `plugin_publish_pipeline_duration_ms`
- **Type**: Histogram
- **Description**: Duration of online publish review pipeline in milliseconds
- **Labels**:
  - `channel`: "online"
  - `status`: "completed" or "sla_exceeded"
- **Alert**: `PublishOnlineSLAExceeded` (warning if > 4 hours, 95th percentile)
- **Query**: `histogram_quantile(0.95, rate(plugin_publish_pipeline_duration_ms_bucket[5m]))`

### SC-003: Offline Approval Duration
- **Metric**: `plugin_offline_approval_duration_minutes`
- **Type**: Histogram
- **Description**: Duration of offline package approval in minutes
- **Labels**:
  - `plugin`: Plugin ID
  - `status`: "completed" or "sla_exceeded"
- **Alert**: `PublishOfflineSLAExceeded` (warning if > 1 day, 95th percentile)
- **Query**: `histogram_quantile(0.95, rate(plugin_offline_approval_duration_minutes_bucket[15m]))`

### SC-004: Rollback Latency
- **Metric**: `plugin_install_rollback_latency_seconds`
- **Type**: Histogram
- **Description**: Time to complete plugin rollback in seconds
- **Labels**:
  - `tenant`: Tenant ID
  - `plugin`: Plugin ID
  - `trigger`: "auto" or "manual"
- **Alert**: `PluginRollbackLatencyExceeded` (critical if > 300 seconds)

### Deployment Status Tracking
- **Metric**: `plugin_deployments_total`
- **Type**: Counter
- **Description**: Total number of deployment attempts by status
- **Labels**:
  - `status`: "pending", "installing", "success", "failed", "rolling_back", "rolled_back"
- **Usage**: Track deployment success/failure rates and retry patterns

## Grafana Dashboard

The following queries can be used in Grafana dashboards:

### Online Review SLA
```promql
# 95th percentile online review duration
histogram_quantile(0.95, rate(plugin_publish_pipeline_duration_ms_bucket[5m]))

# Review count by status
sum by (status) (rate(plugin_publish_pipeline_duration_sum[5m]))
```

### Offline Approval SLA
```promql
# 95th percentile offline approval duration
histogram_quantile(0.95, rate(plugin_offline_approval_duration_minutes_bucket[15m]))

# Count of SLA breaches
sum(rate(plugin_offline_approval_duration_minutes_bucket{status="sla_exceeded"}[15m]))
```

### Rollback Latency
```promql
# Average rollback latency
avg(rate(plugin_install_rollback_latency_seconds_sum[5m])) by (trigger)

# SLA breaches (rollbacks > 5 minutes)
increase(plugin_install_rollback_latency_seconds{le="+Inf"}[5m]) > 300
```

### Deployment Health
```promql
# Success rate
sum(rate(plugin_deployments_total{status="success"}[5m])) /
  sum(rate(plugin_deployments_total[5m]))

# Status distribution
sum by (status) (rate(plugin_deployments_total[5m]))
```

### Capability Registry & Exposure
- **Metric**: `capability_registry_duration_ms`
  - **Type**: Histogram
  - **Labels**: `operation` (init/submit/quota/diff), `result` (success/failure/pending)
  - **Usage**: 观测 CLI/Dev API 交互耗时（70% < 3s，95% < 15s）
  - **Query**:
    ```promql
    histogram_quantile(0.95, rate(capability_registry_duration_ms_bucket[5m]))
    ```
- **Metric**: `capability_exposure_activate_rate`
  - **Type**: Gauge
  - **Labels**: `capability`, `channel`
  - **Usage**: 记录最近窗口暴露成功率（0~1），用于灰度/回滚告警
  - **Query**:
    ```promql
    capability_exposure_activate_rate
    ```
- **CLI Telemetry**: `capability.cli.init_total`, `capability.cli.submit_total`, `capability.cli.quota_total`, `capability.cli.diff_total`
  - 通过 Kibana/Fluentd 收集自 `px-plugin`，字段包含 `capabilityId / tenantId / status / reportPath`
  - 可结合 Prometheus metrics 建立「端到端注册链路」仪表：CLI → Dev API → Host Runtime

示例 Grafana 可视化：

```promql
# 成功率
sum(rate(capability_registry_duration_ms_sum{result="success"}[5m])) /
  sum(rate(capability_registry_duration_ms_count[5m]))

# 暴露通道掉线监控
1 - capability_exposure_activate_rate{channel="rest"}
```

## Alertmanager Configuration

Import `config/alerts/publish-hub.yaml` into Alertmanager to set up:
- Slack notifications for SLA warnings
- PagerDuty escalation for critical rollback failures
- Runbook links for incident response

## Usage

### Accessing Metrics
The `/metrics` endpoint is available at:
```
http://localhost:8080/metrics
```

Metrics are automatically registered when using the observability package:
```go
// In your service initialization
observability.InitMetrics(app)

// Record metrics
observability.RecordRollbackLatency(duration.Seconds(), tenantID, pluginID, "auto")
```

### Metric Buckets

The metrics use exponential histograms with these buckets:
- **Duration (ms)**: 1s, 2s, 4s, 8s, 16s, 32s, 1m, 2m, 4m, 8m, 16m, 32m, 1h, 2h, ~4h
- **Duration (minutes)**: 1, 2, 4, 8, 16, 32, 1h, 2h, 4h, 8h, 16h, 1d, 2d, ~4d
- **Duration (seconds)**: 1, 2, 4, 8, 16, 32, 1m, 2m, 4m, ~8m

This allows for accurate percentile calculations at all relevant time scales.
