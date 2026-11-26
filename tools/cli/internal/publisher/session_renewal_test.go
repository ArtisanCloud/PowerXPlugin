package publisher

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestManagedBootstrapRenewalPreservesEngineRecords(t *testing.T) {
	root := workspace(t)
	e := openTest(t, root)
	p, err := e.Create(input())
	if err != nil {
		t.Fatal(err)
	}
	server, err := Start(context.Background(), e, ServerOptions{Listen: "127.0.0.1:0", Origin: testOrigin})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	load := func() Bootstrap {
		raw, err := e.state.read("tool-session.json", MaxRecordBytes)
		if err != nil {
			t.Fatal(err)
		}
		var value Bootstrap
		if json.Unmarshal(raw, &value) != nil {
			t.Fatal("bootstrap_decode")
		}
		return value
	}
	old := load()
	session := exchange(t, server, old)
	if err = server.RenewBootstrap(); err != nil {
		t.Fatal(err)
	}
	next := load()
	if old.BootstrapToken == next.BootstrapToken || old.InstanceID != next.InstanceID || old.PID != next.PID {
		t.Fatal("renewal_restarted_executor_or_reused_secret")
	}
	decodeData[Plan](t, request(t, server.Info().Endpoint, http.MethodGet, "/v1/plans/"+p.ReleasePlanID, session.Token, ""), http.StatusOK)
	renewed := exchange(t, server, next)
	if renewed.Token == session.Token {
		t.Fatal("session_not_rotated")
	}
	decodeData[Plan](t, request(t, server.Info().Endpoint, http.MethodGet, "/v1/plans/"+p.ReleasePlanID, renewed.Token, ""), http.StatusOK)
	response := request(t, server.Info().Endpoint, http.MethodPost, "/v1/session/renew", renewed.Token, "")
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatal("public_bootstrap_rotation_route_exposed")
	}
}
