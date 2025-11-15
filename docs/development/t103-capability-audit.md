# T103 – Capability 审计与阻断

## 内容概览
- 新增 `.px-plugin/capabilities.json`：记录各 capability 的状态（approved/pending/rejected/unknown）、最后提交时间、备注。
- CLI 在执行 `px-plugin dist`、`px-plugin publish` 时，会读取 manifest + 状态文件；若存在 pending/rejected/unknown 状态将直接阻断并提示先完成审查。
- 每次 `px-plugin capabilities submit` 都会写入 `.px-plugin/audit/<cap>.<timestamp>.log`，方便追踪调试。

## 状态文件格式
```json
{
  "entries": {
    "com.powerx.demo.template.create": {
      "id": "com.powerx.demo.template.create",
      "status": "pending",
      "lastSubmittedAt": "2025-11-30T05:10:00Z",
      "note": "待安全审核"
    }
  }
}
```

## 操作指南
1. `px-plugin capabilities submit --base-url https://dev-api.powerx.local --token ...`
2. 等待审核通过；收到 `status=approved` 后再次运行发布/打包命令。
3. 如需人工编辑，可删除 `.px-plugin/capabilities.json` 再重新提交；不要直接修改为 `approved`。
