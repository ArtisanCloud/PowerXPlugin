-- Run once through the plugin's formal schema migration with search_path
-- pinned to its trusted plugin schema. The worker role never runs migrations.
CREATE TABLE powerx_dataset_source_batches (
    deployment_id text NOT NULL,
    request_uuid uuid NOT NULL,
    grant_uuid uuid NOT NULL,
    plugin_id text NOT NULL,
    installation_revision text NOT NULL,
    tenant_uuid uuid NOT NULL,
    source_binding_uuid uuid NOT NULL,
    deadline_at timestamptz NOT NULL,
    grant_snapshot jsonb NOT NULL,
    outcome text,
    source_receipt_uuid uuid UNIQUE,
    affected_count integer,
    archive_bytes bytea,
    archive_sha256 text NOT NULL DEFAULT '',
    sealed_at timestamptz,
    delivered boolean NOT NULL DEFAULT false,
    PRIMARY KEY (deployment_id, request_uuid),
    CHECK (
        (outcome IS NULL AND source_receipt_uuid IS NULL AND affected_count IS NULL AND archive_bytes IS NULL)
        OR (outcome IS NOT NULL AND outcome IN ('committed','aborted') AND source_receipt_uuid IS NOT NULL
            AND affected_count IS NOT NULL AND affected_count BETWEEN 0 AND 128
            AND (outcome <> 'aborted' OR (affected_count = 0 AND archive_bytes IS NULL)))
    ),
    CHECK (archive_bytes IS NULL OR octet_length(archive_bytes) BETWEEN 1 AND 262144),
    CHECK (archive_sha256 = '' AND archive_bytes IS NULL
        OR (outcome = 'committed' AND archive_sha256 ~ '^[0-9a-f]{64}$'
            AND (delivered AND archive_bytes IS NULL OR NOT delivered AND archive_bytes IS NOT NULL))),
    CHECK (NOT delivered OR sealed_at IS NOT NULL)
);
CREATE INDEX powerx_dataset_source_pending_delivery
    ON powerx_dataset_source_batches (deployment_id, plugin_id, installation_revision, tenant_uuid, source_binding_uuid, (outcome IS NULL), request_uuid)
    WHERE NOT delivered;

CREATE FUNCTION powerx_dataset_guard_source_batch() RETURNS trigger
LANGUAGE plpgsql SET search_path FROM CURRENT AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.outcome IS NULL AND NEW.sealed_at IS NULL AND NOT NEW.delivered THEN
            RETURN NEW;
        END IF;
    ELSIF ROW(NEW.deployment_id, NEW.request_uuid, NEW.grant_uuid, NEW.plugin_id,
              NEW.installation_revision, NEW.tenant_uuid, NEW.source_binding_uuid,
              NEW.deadline_at, NEW.grant_snapshot)
          IS NOT DISTINCT FROM
          ROW(OLD.deployment_id, OLD.request_uuid, OLD.grant_uuid, OLD.plugin_id,
              OLD.installation_revision, OLD.tenant_uuid, OLD.source_binding_uuid,
              OLD.deadline_at, OLD.grant_snapshot) THEN
        IF OLD.outcome IS NULL AND NEW.sealed_at IS NULL AND NOT NEW.delivered THEN
            RETURN NEW;
        ELSIF OLD.outcome IS NOT NULL AND
              ROW(NEW.outcome, NEW.source_receipt_uuid, NEW.affected_count, NEW.archive_sha256)
              IS NOT DISTINCT FROM
              ROW(OLD.outcome, OLD.source_receipt_uuid, OLD.affected_count, OLD.archive_sha256) THEN
            IF OLD.sealed_at IS NULL AND NEW.sealed_at IS NOT NULL
               AND pg_trigger_depth() = 2 AND NOT NEW.delivered
               AND NEW.archive_bytes IS NOT DISTINCT FROM OLD.archive_bytes THEN
                RETURN NEW;
            ELSIF OLD.sealed_at IS NOT NULL AND NEW.sealed_at = OLD.sealed_at
                  AND (NOT OLD.delivered OR NEW.delivered)
                  AND (NEW.archive_bytes IS NOT DISTINCT FROM OLD.archive_bytes
                       OR (NEW.delivered AND NEW.archive_bytes IS NULL)) THEN
                RETURN NEW;
            END IF;
        END IF;
    END IF;
    RAISE EXCEPTION 'DATASET_SOURCE_RECEIPT_CONFLICT' USING ERRCODE = 'check_violation';
END
$$;

CREATE TRIGGER powerx_dataset_guard_source_batch
    BEFORE INSERT OR UPDATE ON powerx_dataset_source_batches
    FOR EACH ROW EXECUTE FUNCTION powerx_dataset_guard_source_batch();

CREATE FUNCTION powerx_dataset_seal_source_batch() RETURNS trigger
LANGUAGE plpgsql SET search_path FROM CURRENT AS $$
DECLARE
    checked_at timestamptz := clock_timestamp();
BEGIN
    IF NEW.outcome = 'committed' AND checked_at >= NEW.deadline_at THEN
        RAISE EXCEPTION 'DATASET_DEADLINE_EXCEEDED' USING ERRCODE = 'check_violation';
    END IF;
    UPDATE powerx_dataset_source_batches
       SET sealed_at = checked_at
     WHERE deployment_id = NEW.deployment_id AND request_uuid = NEW.request_uuid;
    RETURN NULL;
END
$$;

CREATE CONSTRAINT TRIGGER powerx_dataset_seal_source_batch
    AFTER UPDATE OF outcome ON powerx_dataset_source_batches
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW
    WHEN (OLD.outcome IS NULL AND NEW.outcome IS NOT NULL)
    EXECUTE FUNCTION powerx_dataset_seal_source_batch();
