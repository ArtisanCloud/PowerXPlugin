# Error Semantics Matrix

| Scenario | Expected HTTP | Envelope.code | Envelope.message | Required Fields | UI Behavior |
|---|---:|---:|---|---|---|
| Unauthenticated | 401 | 401 | auth required | code,message | redirect login + notice |
| Forbidden | 403 | 403 | permission denied | code,message | keep page + error toast |
| Validation Error | 400 | 400 | invalid params | code,message,details | form inline error |
| Not Found | 404 | 404 | not found | code,message | empty state |
| Server Error | 500 | 500 | internal error | code,message | generic error banner |
