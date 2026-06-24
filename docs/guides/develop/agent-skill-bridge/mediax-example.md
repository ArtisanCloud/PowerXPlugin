# MediaX Skill Template

## Manifest

Use `mediax.video_rebuilder.cn` as the source Skill ID when a media plugin wants PowerX Agent Runtime to create video rebuild tasks.

Required executor contract:

```json
{
  "type": "capability",    "capability": "creation.video_automation.ingest"
}
```

## Executor Result

Long-running media jobs should return `queued` with a stable `task_id`. The executor must keep tenant/user/session identity from `context`, not from input fields.

```json
{
  "success": true,
  "skill_id": "mediax.video_rebuilder.cn",
  "status": "queued",
  "message": "已创建视频重构任务",
  "task_id": "task_xxx",
  "trace_id": "trace_xxx"
}
```
