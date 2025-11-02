package service

import "testing"

func TestPingService(t *testing.T) {
	svc := NewPingService()
	res := svc.Ping()

	if res["status"] != "ok" || len(res) != 1 {
		t.Fatalf("unexpected ping response: %#v", res)
	}
}
