import { describe, expect, it } from 'vitest';
import { upsertNotificationByRevision, type NotificationPayload, type NotificationReadReceiptInput } from '../../../../../framework/frontend/nuxt/framework-client/notifications';
import fixture from '../../../../../docs/contracts/fixtures/business-notification-card.v1.json';

describe('business_notification_card.v1', () => {
  it('preserves the original reference and replaces the same ID by Core revision', () => {
    const initial = structuredClone(fixture.notification) as NotificationPayload;
    const updated = structuredClone(fixture.updated_event) as NotificationPayload;
    let items = upsertNotificationByRevision([initial], updated);
    items = upsertNotificationByRevision(items, initial);
    items = upsertNotificationByRevision(items, updated);
    expect(items).toEqual([updated]);
    expect(items[0]?.metadata?.business_card?.source_revision).toBe(1);
    expect(items[0]?.sourceRevision).toBe(2);
    expect(initial).toEqual(fixture.notification);
  });
  it('preserves the receipt revision map and rejects invalid event versions', () => {
    const receipt: NotificationReadReceiptInput = fixture.read_receipt;
    expect(JSON.parse(JSON.stringify(receipt)).revisions).toEqual(fixture.read_receipt.revisions);
    for (const revision of [0, -1, 1.5, Number.MAX_SAFE_INTEGER + 1]) {
      expect(() => upsertNotificationByRevision([], { ...fixture.updated_event, revision })).toThrow('NOTIFICATION_RESPONSE_INVALID');
    }
  });
});
