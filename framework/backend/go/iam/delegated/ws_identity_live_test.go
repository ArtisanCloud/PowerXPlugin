package delegated

import (
	"context"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	"gopkg.in/yaml.v3"
)

// Opt-in, read-only check against a running Core. It does not issue a proof,
// change grants, create fixtures or read Core SQL. A missing handle is rejected
// after authenticating the real plugin service credential.
func TestLiveWSIdentityRevalidationRejectsUnissuedHandle(t *testing.T) {
	path := os.Getenv("POWERX_WS_IDENTITY_CONFIG")
	seed := os.Getenv("POWERX_WS_IDENTITY_DEV_SEED")
	if path == "" && seed == "" {
		t.Skip("live_identity_config_required")
	}
	var config struct {
		Gateway struct {
			BaseURL string `yaml:"base_url"`
			APIKey  string `yaml:"api_key"`
		} `yaml:"gateway"`
	}
	if seed != "" {
		config.Gateway.BaseURL = os.Getenv("POWERX_WS_IDENTITY_HTTP")
		file, err := parser.ParseFile(token.NewFileSet(), seed, nil, 0)
		if err != nil {
			t.Fatal("dev_seed_read_failed")
		}
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			env, key := "", ""
			for _, element := range literal.Elts {
				pair, ok := element.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				name, ok := pair.Key.(*ast.Ident)
				if !ok {
					continue
				}
				value, ok := pair.Value.(*ast.BasicLit)
				if !ok || value.Kind != token.STRING {
					continue
				}
				text, _ := strconv.Unquote(value.Value)
				if name.Name == "EnvName" {
					env = text
				}
				if name.Name == "Key" {
					key = text
				}
			}
			if env == "POWERX_PLUGIN_API_KEY" {
				config.Gateway.APIKey = key
			}
			return true
		})
	} else {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal("private_config_read_failed")
		}
		if yaml.Unmarshal(raw, &config) != nil {
			t.Fatal("private_config_parse_failed")
		}
	}
	if config.Gateway.APIKey == "" || config.Gateway.BaseURL == "" {
		t.Fatal("private_gateway_config_required")
	}
	client, err := NewCoreClientWithAPIKey(CoreClientConfig{BaseURL: config.Gateway.BaseURL, PluginID: "com.powerx.plugins.base"}, config.Gateway.APIKey)
	if err != nil {
		t.Fatal("service_client_config_failed")
	}
	_, err = client.RevalidateWSIdentity(context.Background(), contracts.WSIdentityQuery{Handle: "pxwsi_v1_zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"})
	var failure *IdentityRequestError
	if !errors.As(err, &failure) {
		t.Fatal("structured_core_response_required")
	}
	// This proves the real service plane and new typed operation were reached,
	// not a valid user's logout/member/permission or installed-plugin acceptance.
	evidence := map[string]any{"scope": "live_core_sdk_api_key_negative_unissued_handle", "http": failure.Status, "expected_http": 401, "code": failure.IAMCode(), "trace_id": failure.TraceID, "request_id": failure.RequestID, "positive_user_connection_verified": false}
	if target := os.Getenv("POWERX_WS_IDENTITY_EVIDENCE"); target != "" {
		raw, _ := json.MarshalIndent(evidence, "", "  ")
		if os.WriteFile(target, append(raw, '\n'), 0600) != nil {
			t.Fatal("evidence_write_failed")
		}
	}
	if failure.Status != 401 || failure.IAMCode() != "IAM_CONTEXT_WS_HANDLE_EXPIRED" {
		t.Fatalf("unexpected_core_result status=%d code=%s trace=%s", failure.Status, failure.IAMCode(), failure.TraceID)
	}
	t.Logf("core_revalidation_reached http=%d code=%s trace=%s", failure.Status, failure.IAMCode(), failure.TraceID)
}
