package server

import (
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
	fwbootstrap "github.com/powerx-plugin/framework/backend/go/bootstrap"
)

var methods = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodOptions,
	http.MethodHead,
}

// RegisterGinRoutes wires every HTTP verb to the underlying gin engine so that
// framework bootstrap.Router can delegate actual handling.
func RegisterGinRoutes(r fwbootstrap.Router, engine *gin.Engine) {
	if r == nil || engine == nil {
		return
	}
	handler := ginHandler(engine)
	for _, method := range methods {
		r.Handle(method, "", handler)
		r.Handle(method, "/*path", handler)
	}
}

func ginHandler(engine *gin.Engine) fwbootstrap.Handler {
	return func(ctx fwbootstrap.Context) {
		writer, req := unwrapHTTP(ctx)
		if writer == nil || req == nil {
			ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to bridge request to gin"})
			return
		}
		engine.ServeHTTP(writer, req)
	}
}

func unwrapHTTP(ctx fwbootstrap.Context) (http.ResponseWriter, *http.Request) {
	rv := reflect.ValueOf(ctx)
	if rv.Kind() != reflect.Ptr {
		return nil, nil
	}
	elem := rv.Elem()
	wField := elem.FieldByName("w")
	reqField := elem.FieldByName("req")
	if !wField.IsValid() || !reqField.IsValid() {
		return nil, nil
	}
	writer, _ := wField.Interface().(http.ResponseWriter)
	req, _ := reqField.Interface().(*http.Request)
	return writer, req
}
