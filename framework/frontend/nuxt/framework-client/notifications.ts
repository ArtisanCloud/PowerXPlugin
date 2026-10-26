export const businessNotificationCardContract = 'business_notification_card.v1' as const;
export const businessNotificationCardContractV2 = 'business_notification_card.v2' as const;

export interface BusinessCardReference {
  contract: typeof businessNotificationCardContract | typeof businessNotificationCardContractV2;
  source_plugin: string;
  business_type: string;
  delivery_key: string;
  source_revision: number;
}
export interface NotificationNavigation { label: string; path: string }
export interface BusinessCardPresentation {
  contract: typeof businessNotificationCardContract;
  revision: number; title: string; summary: string; updated_at: string;
  unread_change_count: number; image?: { image_key: string; alt: string };
}
export interface BusinessCardPresentationSource {
  contract: BusinessCardReference['contract']; source_plugin: string;
  presentation_url: string; preview_url_template: string;
}
export interface NotificationAction {
  id: string; label: string; type: string; action: string; params: Record<string, string>;
}
export interface NotificationPayload {
  id: string;
  revision: number;
  sourceRevision?: number;
  title: string;
  content: string;
  type: string;
  category: string;
  isRead: boolean;
  isImportant: boolean;
  createdAt: string;
  updatedAt: string;
  userId?: string;
  metadata?: Record<string, unknown> & { business_card?: BusinessCardReference };
  actions?: NotificationAction[];
}
export interface NotificationUpdatedEvent extends NotificationPayload {
  eventType: 'notification_updated'; payload_version: 1; tenant_uuid: string;
}
export interface NotificationReadReceiptInput {
  member_uuid: string;
  notification_uuids: string[];
  revisions?: Record<string, number>;
}
export interface NotificationReadReceipt extends NotificationReadReceiptInput {
  contract: 'notification_read_receipt.v1';
}
export interface InvalidateCardInput {
  member_uuid: string; notification_uuid: string; source_revision: number;
  reason?: 'content_changed' | 'access_changed' | 'presentation_changed'; card_reference?: BusinessCardReference;
}
export interface InvalidateCardResult {
  notification_uuid: string; revision: number; source_revision: number; changed: boolean;
}

const positiveRevision = (revision: number | undefined) => Number.isSafeInteger(revision) && (revision ?? 0) > 0;

export type CardRenditionStatus = 'not_requested' | 'waiting_upload' | 'waiting_source' | 'queued' | 'processing' | 'ready' | 'failed' | 'scheduling_failed' | 'unsupported' | 'unavailable';
export type CardRendition<V extends 'thumbnail_64' | 'preview'> = {
  variant: V; error_code?: string; retryable?: boolean;
} & ({ status: 'ready'; image_key: string } | { status: Exclude<CardRenditionStatus, 'ready'>; image_key?: never });
export interface BusinessCardPresentationV2 extends Omit<BusinessCardPresentation, 'contract' | 'image'> {
  contract: typeof businessNotificationCardContractV2;
  image?: { alt: string; thumbnail: CardRendition<'thumbnail_64'>; preview: CardRendition<'preview'> };
}
export type BusinessCardPresentationSnapshot = BusinessCardPresentation | BusinessCardPresentationV2;

export function validateBusinessCardPresentation<T extends BusinessCardPresentationSnapshot>(value: T): T {
  const invalid = () => { throw new Error('NOTIFICATION_PRESENTATION_INVALID'); };
  if (!value || !positiveRevision(value.revision) || !Number.isSafeInteger(value.unread_change_count) || value.unread_change_count < 0 || typeof value.title !== 'string' || !value.title || value.title.length > 255 || typeof value.summary !== 'string' || value.summary.length > 4000 || !Number.isFinite(Date.parse(value.updated_at))) invalid();
  if (value.contract !== businessNotificationCardContract && value.contract !== businessNotificationCardContractV2) invalid();
  if (Object.keys(value).some(key => !["contract","revision","title","summary","updated_at","unread_change_count","image"].includes(key))) invalid();
  if (value.image === undefined) return value;
  if (!value.image || typeof value.image !== "object") invalid();
  const image = value.image;
  if (typeof image.alt !== 'string' || image.alt.length > 255) invalid();
  const token = /^[A-Za-z0-9_-]{1,256}$/;
  if (value.contract === businessNotificationCardContract) {
    if (typeof (image as BusinessCardPresentation['image'])!.image_key !== "string" || !token.test((image as BusinessCardPresentation['image'])!.image_key) || Object.keys(image).some(key => !['alt','image_key'].includes(key))) invalid();
  } else {
    const v2 = value.image!;
    if (Object.keys(v2).some(key => !['alt','thumbnail','preview'].includes(key))) invalid();
    for (const [name, variant] of [['thumbnail','thumbnail_64'], ['preview','preview']] as const) {
      const item = v2[name];
      if (!item || item.variant !== variant || !['not_requested','waiting_upload','waiting_source','queued','processing','ready','failed','scheduling_failed','unsupported','unavailable'].includes(item.status) || Object.keys(item).some(key => !['variant','status','image_key','error_code','retryable'].includes(key))) invalid();
      if (item.status === 'ready' ? typeof item.image_key !== 'string' || !token.test(item.image_key) : 'image_key' in item) invalid();
      if ((item.error_code !== undefined && (typeof item.error_code !== 'string' || item.error_code.length > 128)) || (item.retryable !== undefined && typeof item.retryable !== 'boolean')) invalid();
    }
  }
  return value;
}

/** A pure merge for existing realtime consumers. It opens no connection and
 * never interprets a notification receipt as business-content reading. */
export function upsertNotificationByRevision<T extends NotificationPayload>(items: readonly T[], incoming: T): T[] {
  if (!incoming.id || !positiveRevision(incoming.revision) ||
      (incoming.sourceRevision !== undefined && !positiveRevision(incoming.sourceRevision)) ||
      (incoming.metadata?.business_card && !positiveRevision(incoming.sourceRevision))) {
    throw new Error('NOTIFICATION_RESPONSE_INVALID');
  }
  const index = items.findIndex(item => item.id === incoming.id);
  if (index < 0) return [...items, incoming];
  if (items[index]!.revision >= incoming.revision) return [...items];
  const next = [...items]; next[index] = incoming;
  return next;
}
