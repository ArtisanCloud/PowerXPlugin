# PowerX Plugin Framework Client

PowerX iframe bridge v2 client, version `0.0.12`. This release includes structured authentication error classification, verified host-session handling, Router navigation, and WS/SSE session binding. Go service authentication is provided by the separately versioned backend Framework.

## Embedded authentication

Use the Core consumer on the iframe's own `window.PowerXBridgeClient`. The protocol must be v2. Never overwrite that object or register a parallel authentication message listener.

```ts
import {
  initPowerXBridge,
  getPowerXHostSession,
  createPowerXRouterNavigator,
} from '@artisan-cloud/plugin-framework-client';

const session = getPowerXHostSession(pluginId);
const bridge = initPowerXBridge({
  pluginId,
  onAuthToken: message => session.apply(message, {
    readContext: (candidate, signal) => readActualCoreContext(candidate, signal),
    validateContext: (context, candidate) => validateAndReturnIdentity(context, candidate),
    commitContext: (context, verified) => commitPluginContext(context, verified),
    clearContext: () => clearPluginOwnedCaches(),
  }),
  navigator: createPowerXRouterNavigator(router, appBaseURL, window.location.origin),
});
const stopRequester = session.setRequester(() => bridge.consumer.requestAuth());
// Dispose both registrations at the corresponding application lifecycle boundary.
```

`validateContext` returns `{ tenantUuid, memberUuid }` from verified Core context. Check enabled user status, current tenant and member UUID, membership, permission_codes and Root rules. Do not commit before validation. Original message fields remain in the snapshot; verified identity is a separate immutable field. Ordinary HTTP, WS and SSE use only verified state. Context validation requests use a current candidate snapshot for the fixed context endpoint.

Consume accessToken only. Do not use a refreshToken, host storage, backup Token, Cookie or scoped login/refresh/logout endpoint. The Core consumer sends applied/rejected after the returned Promise settles. A previous session cannot retry through a later login. 403 does not refresh. A cleared session can recover through an explicit new synchronization action.

Both `createPowerXApiFetchClient` and the native `usePluginApi` entrypoint use the shared session. Only HTTP 401 with a structured user-token expiry code (`IAM_CONTEXT_USER_SESSION_EXPIRED`, or the standalone user-token codes) permits one retry after a verified snapshot is received. `IAM_CONTEXT_USER_SESSION_INVALID` retires the plugin consumer state; service/proof failures do not clear it. Proof/service credential failures, ambiguous `IAM_UNAUTHORIZED`, unclassified 401, and 403 do not request host authentication. The native client retains the actual error response, body and trace headers. Embedded baseURL is `/_p/<pluginId>/api/v1`. Set Nuxt app.baseURL/NUXT_APP_BASE_URL to the actual plugin admin mount. Router navigation preserves query/hash, waits for Router and handles cancellation without a document reload.

## Standalone

Do not require the iframe consumer outside PowerX. Continue using the plugin's own authentication adapter. Frontend embedding is independent of backend Local/Delegated Provider mode.

## Adoption and verification

Copied authentication code in other plugins must be updated with the package; upgrading the dependency alone is insufficient. See the repository's `docs/contracts/plugin-session-bridge-v2.md` and `docs/guides/features/plugin-session-bridge-v2/guide.md` for Skeleton wiring and acceptance steps.

Contract tests use the Core injection script as a Node VM fixture and actual Vue Router memory history. These are not live browser acceptance. Real login, first refresh, multiple tabs, WS, images/downloads and browser history must be validated with the paired Core/plugin builds before declaring the integration accepted.
