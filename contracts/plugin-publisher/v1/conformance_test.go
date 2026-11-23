package pluginpublisher

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func fixture(t *testing.T, name string, into any) {
	t.Helper()
	raw, err := Bundle.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, into); err != nil {
		t.Fatal(err)
	}
}

func TestSharedWireAndSemanticCases(t *testing.T) {
	var data struct {
		Cases []struct {
			Name, Definition, Raw string
			Data                  json.RawMessage
			Valid                 bool
		}
	}
	fixture(t, "schema-cases.json", &data)
	for _, row := range data.Cases {
		t.Run(row.Name, func(t *testing.T) {
			raw := row.Data
			if row.Raw != "" {
				raw = []byte(row.Raw)
			}
			_, err := Validate(row.Definition, raw)
			if (err == nil) != row.Valid {
				t.Fatalf("valid=%v; error=%v", row.Valid, err)
			}
		})
	}
}

func TestEveryDefinitionCompiles(t *testing.T) {
	initOnce.Do(initialize)
	if initErr != nil {
		t.Fatal(initErr)
	}
	var schema struct {
		Defs map[string]json.RawMessage `json:"$defs"`
	}
	fixture(t, "protocol.schema.json", &schema)
	for definition := range schema.Defs {
		compileMu.Lock()
		_, err := compiler.Compile(schemaID + "#/$defs/" + definition)
		compileMu.Unlock()
		if err != nil {
			t.Fatalf("definition %s: %v", definition, err)
		}
	}
}

func TestCrossLanguageCanonicalRequests(t *testing.T) {
	var data struct {
		Vectors []struct {
			Name, Type, Canonical, SHA256 string
			Input                         json.RawMessage
		}
	}
	fixture(t, "canonical-cases.json", &data)
	for _, row := range data.Vectors {
		t.Run(row.Name, func(t *testing.T) {
			canonical, digest, err := Fingerprint(row.Type, row.Input)
			if err != nil {
				t.Fatal(err)
			}
			if canonical != row.Canonical || digest != row.SHA256 {
				t.Fatalf("cross-language fingerprint differs: canonical=%q sha256=%s", canonical, digest)
			}
			if row.Type == "package-upload" {
				var changed map[string]any
				_ = json.Unmarshal(row.Input, &changed)
				changed["sha256"] = strings.Repeat("d", 64)
				raw, _ := json.Marshal(changed)
				_, other, err := Fingerprint(row.Type, raw)
				if err != nil || other == digest {
					t.Fatal("changed package bytes must change request identity", err)
				}
			}
		})
	}
	if data.Vectors[0].SHA256 != data.Vectors[1].SHA256 || data.Vectors[4].SHA256 != data.Vectors[5].SHA256 {
		t.Fatal("order/normalization equivalence vectors differ")
	}
	if data.Vectors[0].SHA256 == data.Vectors[2].SHA256 || data.Vectors[4].SHA256 == data.Vectors[6].SHA256 {
		t.Fatal("changed business content must have a different identity")
	}
}

func TestTrustedConsoleLinks(t *testing.T) {
	var data struct {
		Cases []struct {
			Name, Base, Target string
			Valid              bool
		}
	}
	fixture(t, "link-cases.json", &data)
	for _, row := range data.Cases {
		t.Run(row.Name, func(t *testing.T) {
			if TrustedMarketLink(row.Base, row.Target) != row.Valid {
				t.Fatal("trusted console link decision differs")
			}
		})
	}
}

func TestSharedRecoveryDecisions(t *testing.T) {
	var data struct {
		Cases []struct {
			Name, Action string
			Context      json.RawMessage
		}
	}
	fixture(t, "recovery-cases.json", &data)
	for _, row := range data.Cases {
		t.Run(row.Name, func(t *testing.T) {
			action, err := RecoveryAction(row.Context)
			if err != nil || action != row.Action {
				t.Fatalf("action=%s expected=%s err=%v", action, row.Action, err)
			}
		})
	}
}

func TestSharedRawEncodingCases(t *testing.T) {
	var cases struct {
		Cases []struct {
			Name, Hex string
			MaxBytes  int `json:"max_bytes"`
			Valid     bool
		}
	}
	fixture(t, "encoding-cases.json", &cases)
	for _, c := range cases.Cases {
		t.Run(c.Name, func(t *testing.T) {
			raw, err := hex.DecodeString(c.Hex)
			if err != nil {
				t.Fatal(err)
			}
			_, err = strictJSONWithLimit(raw, c.MaxBytes)
			if (err == nil) != c.Valid {
				t.Fatalf("valid=%v err=%v", c.Valid, err)
			}
		})
	}
}

func TestStrictEncodingAndConcurrency(t *testing.T) {
	for _, raw := range [][]byte{
		{'"', 0xff, '"'},
		[]byte(`{"version":"1.2.0","version":"1.2.0"}`),
		[]byte(`"\ud800"`),
		[]byte(`"\udc00"`),
		[]byte(strings.Repeat("[", 66) + "0" + strings.Repeat("]", 66)),
		bytes.Repeat([]byte(" "), 65537),
	} {
		if _, err := strictJSON(raw); err == nil {
			t.Fatal("invalid strict JSON accepted")
		}
	}
	for range 8 {
		t.Run("parallel schema consumption", func(t *testing.T) {
			t.Parallel()
			if _, err := Validate("PluginID", []byte(`"com.example.demo"`)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRequestAndResponseBudgetsDiffer(t *testing.T) {
	var examples struct {
		Cases []struct {
			Definition string
			Data       json.RawMessage
		}
	}
	fixture(t, "examples.json", &examples)
	for _, row := range examples.Cases {
		if row.Definition != "DiscoveryResponse" {
			continue
		}
		padded := append(append([]byte{}, row.Data...), bytes.Repeat([]byte(" "), 65536)...)
		if _, err := Validate(row.Definition, padded); err != nil {
			t.Fatal("response is allowed to exceed the JSON request budget", err)
		}
		tooLarge := append(append([]byte{}, row.Data...), bytes.Repeat([]byte(" "), 8388608)...)
		if _, err := Validate(row.Definition, tooLarge); err == nil {
			t.Fatal("response byte budget not enforced")
		}
	}
	request := append([]byte(`{"name":"Local publisher","plugin_ids":["com.example.demo"],"scopes":["plugins:read"]}`), bytes.Repeat([]byte(" "), 65536)...)
	if _, err := Validate("CreateTokenRequest", request); err == nil {
		t.Fatal("request budget not enforced")
	}
}
