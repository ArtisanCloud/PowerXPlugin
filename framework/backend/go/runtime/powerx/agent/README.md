# PowerX Agent Client

`runtime/powerx/agent` wraps PowerX Agent Runtime access for plugins.

The client owns non-stream invoke calls, SSE and WebSocket event decoding, delegated bearer authentication, diagnostics, and standard error mapping. Plugin business code should not reimplement Agent stream protocol parsing.
