from fastapi import Request


def attach_host_context(request: Request) -> None:
    request.state.host_proxy = request.headers.get("x-powerx-proxy")
    request.state.plugin_id = request.headers.get("x-plugin-id")
