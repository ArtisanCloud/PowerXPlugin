package customer

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	contactfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/contactfw"
	"github.com/gin-gonic/gin"
)

func TestContactHandlerPreservesPrimaryBusinessErrorAndTrace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		code   contactfw.ErrorCode
		status int
	}{
		{contactfw.CodePrimaryExpectedRequired, 400},
		{contactfw.CodePrimaryChanged, 409},
		{contactfw.CodePrimaryOperationFailed, 500},
		{contactfw.CodePrimaryChanged, 502},
	} {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest("POST", "/", nil)
		upstream := &gateway.InvocationError{StatusCode: tc.status, TraceID: "fixture-trace", UpstreamRequestID: "fixture-request"}
		NewContactHandler(nil).respondError(ctx, contactfw.NewError(tc.code, upstream))
		var body struct {
			Error struct {
				Code    string            `json:"code"`
				Details map[string]string `json:"details"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if rec.Code != tc.status || body.Error.Code != string(tc.code) || body.Error.Details["trace_id"] != "fixture-trace" || body.Error.Details["upstream_request_id"] != "fixture-request" {
			t.Fatalf("CONTACT_PRIMARY_HTTP_RESPONSE_INVALID: status=%d body=%s", rec.Code, rec.Body.String())
		}
	}
}
