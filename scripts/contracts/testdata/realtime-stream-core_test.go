// Loaded into the Core test package by verify-realtime-stream-core.py.
package capability_registry

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	cap "github.com/ArtisanCloud/PowerX/internal/service/capability_registry"
	runtimehost "github.com/ArtisanCloud/PowerX/internal/service/runtime_host"
	cm "github.com/ArtisanCloud/PowerX/pkg/corex/db/persistence/model/capability_registry"
	im "github.com/ArtisanCloud/PowerX/pkg/corex/db/persistence/model/iam"
	gw "github.com/ArtisanCloud/PowerX/pkg/corex/db/persistence/model/integration_gateway"
	"github.com/ArtisanCloud/PowerX/pkg/dto"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

type frameworkRealtimeDelegate struct {
	invoker  *runtimehost.RealtimeInvoker
	failures chan error
}

func (d *frameworkRealtimeDelegate) InvokeCoreCapability(ctx context.Context, in cap.CoreCapabilityInvokeInput) (map[string]any, error) {
	return d.invoker.InvokeCoreCapability(ctx, in)
}

func (d *frameworkRealtimeDelegate) InvokeCoreCapabilityStream(ctx context.Context, in cap.CoreCapabilityInvokeInput) (io.ReadCloser, error) {
	stream, err := d.invoker.InvokeCoreCapabilityStream(ctx, in)
	if err != nil {
		d.failures <- err
	}
	return stream, err
}

func TestFrameworkRealtimeCredentialTenantInterop(t *testing.T) {
	for _, kind := range []string{"gateway", "capability"} {
		for _, scenario := range []string{"ready", "wrong_credential", "cross_tenant", "missing_grant", "live_revocation", "cancel"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				delegate := &frameworkRealtimeDelegate{failures: make(chan error, 1)}
				f := newInvocationHTTPFixture(t, false, delegate)
				// Retarget the isolated fixture's catalog and grant to the real realtime
				// invoker. Authentication, tenant guards and grant evaluation are Core code.
				binding, err := json.Marshal([]cm.ProtocolBinding{{Channel: "core_internal", Method: "INVOKE", Endpoint: runtimehost.RealtimeEndpoint, AuthType: "api_key_or_sts", ActorContext: "service_actor", ResourceScope: "tenant"}})
				require.NoError(t, err)
				require.NoError(t, f.db.Model(&cm.CapabilityRecord{}).Where("capability_id=?", f.capID).Updates(map[string]any{"capability_id": runtimehost.RealtimeStream, "protocols": datatypes.JSON(binding)}).Error)
				require.NoError(t, f.db.Model(&cm.CapabilityRegistration{}).Where("capability_id=?", f.capID).Update("capability_id", runtimehost.RealtimeStream).Error)
				require.NoError(t, f.db.Model(&cm.AdapterEndpoint{}).Where("capability_id=?", f.capID).Updates(map[string]any{"capability_id": runtimehost.RealtimeStream, "transport_type": "core_internal", "endpoint": runtimehost.RealtimeEndpoint, "labels": datatypes.JSON(`{"method":"INVOKE"}`)}).Error)
				meta, err := json.Marshal(map[string]any{"capability_id": runtimehost.RealtimeStream, "api_key_explicit": true, "api_key": map[string]string{"scope": "_scope.fixture.service", "action": "read", "resource_type": "api", "resource_pattern": runtimehost.RealtimeStream, "effect": "allow"}})
				require.NoError(t, err)
				require.NoError(t, f.db.Model(&im.Permission{}).Where("resource=?", f.capID).Updates(map[string]any{"resource": runtimehost.RealtimeStream, "meta": datatypes.JSON(meta)}).Error)
				require.NoError(t, f.db.Model(&gw.IntegrationGatewayAPIKeyPermission{}).Where("api_key_uuid=?", f.key.UUID).Update("resource_pattern", runtimehost.RealtimeStream).Error)
				mini := miniredis.RunT(t)
				redisClient := redis.NewClient(&redis.Options{Addr: mini.Addr()})
				t.Cleanup(func() { require.NoError(t, redisClient.Close()) })
				delegate.invoker = runtimehost.NewRealtimeInvoker(f.db, runtimehost.NewService(f.db, redisClient, "framework-realtime-interop"))
				server := httptest.NewServer(f.engine)
				t.Logf("CORE_FIXTURE pid=%d parent_pid=%d http=%s redis=%s", os.Getpid(), os.Getppid(), server.Listener.Addr(), mini.Addr())
				t.Cleanup(func() {
					address := server.Listener.Addr().String()
					server.Close()
					connection, dialErr := net.DialTimeout("tcp", address, 100*time.Millisecond)
					if connection != nil {
						_ = connection.Close()
					}
					require.Error(t, dialErr)
				})
				secret, tenant := f.secret, apiKeyCatalogTestTenant
				if scenario == "wrong_credential" {
					secret = "pxk_invalid_fixture"
				}
				if scenario == "cross_tenant" {
					tenant = "00000000-0000-4000-8000-000000000099"
				}
				revoke := func() {
					require.NoError(t, f.db.Where("api_key_uuid=?", f.key.UUID).Delete(&gw.IntegrationGatewayAPIKeyPermission{}).Error)
				}
				if scenario == "missing_grant" {
					revoke()
				}
				var stream func(context.Context, hostcontract.StreamInput, func(hostcontract.StreamEvent) error) error
				if kind == "gateway" {
					client, err := gateway.NewClient(gateway.Config{BaseURL: server.URL, AuthScheme: "apikey", APIKey: secret})
					require.NoError(t, err)
					t.Cleanup(func() { _ = client.Close() })
					stream = client.InvokeStream
				} else {
					client, err := capability.NewClient(capability.Config{BaseURL: server.URL, AuthScheme: "apikey", APIKey: secret}, nil)
					require.NoError(t, err)
					stream = client.InvokeStream
				}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				stop := errors.New("FIXTURE_STREAM_STOP")
				var received []string
				err = stream(ctx, hostcontract.StreamInput{CapabilityID: runtimehost.RealtimeStream, PreferredProtocol: "core_internal", TenantUUID: apiKeyCatalogTestTenant, Body: map[string]any{"contract_version": runtimehost.RealtimeContract, "tenant_key": tenant, "namespace": "fixture.realtime", "operation": "subscribe", "cursor": "0-0"}, RequestID: "framework-request", TraceID: "framework-trace"}, func(event hostcontract.StreamEvent) error {
					received = append(received, event.Event)
					require.NotEmpty(t, event.TraceID)
					t.Logf("CORE_FRAME event=%s trace=%s", event.Event, event.TraceID)
					var data map[string]any
					require.NoError(t, json.Unmarshal([]byte(event.Data), &data))
					if event.Event == "error" {
						require.Equal(t, "REALTIME_GRANT_REQUIRED", data["code"])
						return stop
					}
					require.Equal(t, "ready", event.Event)
					require.Equal(t, apiKeyCatalogTestTenant, data["tenant_key"])
					require.Equal(t, "fixture.realtime", data["namespace"])
					require.Equal(t, "0-0", data["cursor"])
					if scenario == "live_revocation" {
						revoke()
						return nil
					}
					if scenario == "cancel" {
						cancel()
						return nil
					}
					return stop
				})
				switch scenario {
				case "ready":
					require.ErrorIs(t, err, stop)
					require.Equal(t, []string{"ready"}, received)
				case "live_revocation":
					require.ErrorIs(t, err, stop)
					require.Equal(t, []string{"ready", "error"}, received)
				case "cancel":
					require.ErrorIs(t, err, context.Canceled)
					require.Equal(t, []string{"ready"}, received)
				default:
					require.Empty(t, received)
					status := 0
					var gatewayErr *gateway.InvocationError
					var capabilityErr *capability.HTTPError
					if errors.As(err, &gatewayErr) {
						status = gatewayErr.StatusCode
					} else if errors.As(err, &capabilityErr) {
						status = capabilityErr.StatusCode
					}
					if scenario == "cross_tenant" {
						// Core currently maps REALTIME_INVALID_ARGUMENT to HTTP 502.
						// Verify the real scope denial independently of that mapping gap.
						require.GreaterOrEqual(t, status, 400)
						require.Less(t, status, 600)
						select {
						case failure := <-delegate.failures:
							require.Equal(t, 400, dto.StatusCode(failure))
							require.Equal(t, "REALTIME_INVALID_ARGUMENT", dto.CodeOf(failure))
						case <-ctx.Done():
							t.Fatal("CORE_SCOPE_DENIAL_MISSING")
						}
						t.Logf("CORE_SCOPE_DENIAL http_status=%d", status)
					} else {
						want := map[string]int{"wrong_credential": 401, "missing_grant": 403}[scenario]
						require.Equal(t, want, status)
					}
				}
			})
		}
	}
}
