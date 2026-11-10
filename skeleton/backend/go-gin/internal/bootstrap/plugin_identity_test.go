package bootstrap

import (
	"context"
	"errors"
	"testing"

	fwmanifest "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/manifest"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
)

func TestPluginIdentityFailsBeforeDatabase(t *testing.T) {
	for _, field := range []string{"POWERX_PLUGIN_ID", "POWERX_SECURITY_JWT_AUDIENCE"} {
		t.Run(field, func(t *testing.T) {
			t.Setenv("POWERX_PLUGIN_ID", app.PluginID)
			t.Setenv("POWERX_SECURITY_JWT_AUDIENCE", "plugin:"+app.PluginID)
			t.Setenv(field, "com.example.other")
			_, err := BootstrapPlugin(context.Background(), nil)
			var identity *fwmanifest.IdentityError
			if !errors.As(err, &identity) || identity.Code != "PLUGIN_ID_MISMATCH" {
				t.Fatal(err)
			}
		})
	}
}
