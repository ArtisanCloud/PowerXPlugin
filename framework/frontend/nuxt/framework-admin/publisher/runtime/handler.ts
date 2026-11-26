import {
  defineEventHandler,
  getHeaders,
  getRequestURL,
  getRequestWebStream,
  setHeader,
  setResponseStatus
} from 'h3'
import {
  maxPublisherRequestBytes,
  PublisherProxyError
} from '../proxy'
import { managedPublisherOrigin, type createPublisherManager } from '../manager'
import { publisherAPIPath } from '../types'

export function createPublisherDevHandler(manager: ReturnType<typeof createPublisherManager>) {
  return defineEventHandler(async (event) => {
    setHeader(event, 'Cache-Control', 'no-store')
    setHeader(event, 'X-Content-Type-Options', 'nosniff')
    try {
      const url = getRequestURL(event)
      const request = {
        path:
          (url.pathname.startsWith(publisherAPIPath)
            ? url.pathname.slice(publisherAPIPath.length)
            : url.pathname) + url.search,
        method: event.method,
        headers: new Headers(getHeaders(event) as Record<string, string>),
        remoteAddress: event.node.req.socket.remoteAddress || '',
        localPort: event.node.req.socket.localPort,
        secure: (event.node.req.socket as { encrypted?: boolean }).encrypted === true,
        body: undefined as Uint8Array | undefined
      }
      managedPublisherOrigin(request)
      if (Number(request.headers.get('content-length') || 0) > maxPublisherRequestBytes)
        throw new PublisherProxyError('PUBLISHER_REQUEST_TOO_LARGE', 413)
      const stream = getRequestWebStream(event)
      if (stream) {
        const reader = stream.getReader()
        const chunks: Uint8Array[] = []
        let length = 0
        try {
          for (;;) {
            const { done, value } = await reader.read()
            if (done) break
            length += value.byteLength
            if (length > maxPublisherRequestBytes)
              throw new PublisherProxyError('PUBLISHER_REQUEST_TOO_LARGE', 413)
            chunks.push(value)
          }
        } finally {
          reader.releaseLock()
        }
        request.body = Buffer.concat(chunks)
      }
      const result = await manager.request(request)
      setResponseStatus(event, result.status)
      return result.body
    } catch (error) {
      const failure =
        error instanceof PublisherProxyError
          ? error
          : new PublisherProxyError('PUBLISHER_PROXY_UNAVAILABLE', 503)
      setResponseStatus(event, failure.status)
      return failure.body()
    }
  })
}
