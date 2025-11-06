package observability

import (
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
)

func TestInitMetricsAndTracing(t *testing.T) {
	app := bootstrap.NewApp(nil)

	if err := InitMetrics(app); err != nil {
		t.Fatalf("InitMetrics returned error: %v", err)
	}
	if err := InitTracing(app); err != nil {
		t.Fatalf("InitTracing returned error: %v", err)
	}
}
