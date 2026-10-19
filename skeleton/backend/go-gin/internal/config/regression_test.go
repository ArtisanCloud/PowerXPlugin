package config

import "testing"

func TestRegressionConfigExplicitEnvironment(t *testing.T) {
	cfg := RegressionConfig{}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, env := range []string{"", "production"} {
		cfg = RegressionConfig{Enabled: true, Environment: env}
		if err := cfg.Validate(); err == nil {
			t.Fatal(env)
		}
	}
	cfg = RegressionConfig{Enabled: true, Environment: "test", Profiles: []RegressionProfileConfig{{UUID: "44444444-4444-4444-8444-444444444444", Key: "controlled", Kind: "controlled", TitleKey: "regression.profiles.controlled", FixtureNamespace: "regression"}}}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	cfg.Profiles = append(cfg.Profiles, cfg.Profiles[0])
	if err := cfg.Validate(); err == nil {
		t.Fatal("REGRESSION_INVALID_REQUEST")
	}
}
