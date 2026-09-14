package aisettings

import (
	"context"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
)

type catalogInvokerStub struct {
	request gateway.InvokeRequest
	payload any
}

func (s *catalogInvokerStub) Invoke(_ context.Context, request gateway.InvokeRequest) (*gateway.Response, error) {
	s.request = request
	return &gateway.Response{Data: map[string]any{"payload": s.payload}}, nil
}

func TestCatalogProvidersUsesPublishedCoreCapabilityAndRoute(t *testing.T) {
	stub := &catalogInvokerStub{payload: []CatalogProvider{{"id": "ollama", "name": "Ollama (Local)"}}}
	client, err := NewClient(Config{Invoker: stub})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.CatalogProviders(context.Background(), "llm", "request-1"); err != nil {
		t.Fatal(err)
	}
	if stub.request.CapabilityID != "com.corex.ai.catalog.providers.read" {
		t.Fatalf("capability=%q", stub.request.CapabilityID)
	}
	payload, ok := stub.request.Payload.(restPayload)
	if !ok || payload.Endpoint != "/api/v1/admin/agents/providers" || payload.Query["env"] != "dev" || payload.Query["modality"] != "llm" {
		t.Fatalf("payload=%#v", stub.request.Payload)
	}
}

func TestCatalogProvidersReadsCoreRESTEnvelope(t *testing.T) {
	stub := &catalogInvokerStub{payload: map[string]any{
		"code": 200,
		"data": map[string]any{
			"providers": []CatalogProvider{{"ID": "ollama", "Name": "Ollama (Local)"}},
		},
	}}
	client, err := NewClient(Config{Invoker: stub})
	if err != nil {
		t.Fatal(err)
	}
	items, err := client.CatalogProviders(context.Background(), "llm", "request-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0]["ID"] != "ollama" {
		t.Fatalf("items=%#v", items)
	}
}

func TestCatalogModelsUsesPublishedCoreCapabilityAndRoute(t *testing.T) {
	stub := &catalogInvokerStub{payload: map[string]any{"models": []string{"qwen3:8b"}}}
	client, err := NewClient(Config{Invoker: stub})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.CatalogModels(context.Background(), "llm", "ollama", "", "request-1"); err != nil {
		t.Fatal(err)
	}
	if stub.request.CapabilityID != "com.corex.ai.catalog.models.read" {
		t.Fatalf("capability=%q", stub.request.CapabilityID)
	}
	payload, ok := stub.request.Payload.(restPayload)
	if !ok || payload.Endpoint != "/api/v1/admin/agents/models" || payload.Query["env"] != "dev" || payload.Query["modality"] != "llm" || payload.Query["provider"] != "ollama" {
		t.Fatalf("payload=%#v", stub.request.Payload)
	}
}
