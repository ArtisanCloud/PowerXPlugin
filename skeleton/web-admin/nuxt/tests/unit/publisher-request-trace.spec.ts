import { expect, it } from 'vitest'
import {
  createPublisherClient,
  PublisherClientError
} from '../../../../../framework/frontend/nuxt/framework-admin/publisher/client'
const uuid = (n: number) =>
  `00000000-0000-4000-8000-${String(n).padStart(12, '0')}`
const response =
  (data: unknown, status = 200) =>
  async () =>
    new Response(JSON.stringify(data), {
      status,
      headers: { 'Content-Type': 'application/json' }
    })
it('preserves the canonical market request identifier without upstream prose', async () => {
  const client = createPublisherClient(
    'http://localhost:3131',
    response(
      {
        error: {
          code: 'PUBLISHER_MARKET_REJECTED',
          params: { request_id: uuid(1), message: 'fixture_private' }
        }
      },
      403
    ) as typeof fetch
  )
  await expect(
    client.prepareSubmission(uuid(2), uuid(3))
  ).rejects.toMatchObject({
    code: 'PUBLISHER_MARKET_REJECTED',
    requestID: uuid(1)
  })
  const invalid = createPublisherClient(
    'http://localhost:3131',
    response(
      {
        error: {
          code: 'PUBLISHER_MARKET_REJECTED',
          params: { request_id: 'fixture_private' }
        }
      },
      403
    ) as typeof fetch
  )
  await expect(
    invalid.prepareSubmission(uuid(2), uuid(3))
  ).rejects.toMatchObject({
    code: 'PUBLISHER_PROXY_RESPONSE_INVALID',
    requestID: ''
  })
})
it('rejects request records bound to a different task', async () => {
  const item = {
    schema: 'powerx.plugin-publisher-request/v1',
    task_kind: 'submissions',
    task_id: uuid(1),
    operation_id: uuid(2),
    client_request_id: uuid(3),
    server_request_id: uuid(4),
    method: 'POST',
    response_type: 'SubmissionReceiptResponse',
    http_status: 201,
    started_at: new Date().toISOString()
  }
  const client = createPublisherClient(
    'http://localhost:3131',
    response({
      data: { items: [item], page: 1, size: 20, total: 1 }
    }) as typeof fetch
  )
  expect(
    (await client.requestTraces('submissions', uuid(1))).items[0]
      .server_request_id
  ).toBe(uuid(4))
  await expect(
    client.requestTraces('submissions', uuid(5))
  ).rejects.toBeInstanceOf(PublisherClientError)
})
