package route

import (
	"net/http"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	v2 "github.com/IceWhaleTech/CasaOS-LocalStorage/route/v2"
	"github.com/deepmap/oapi-codegen/pkg/middleware"
	"github.com/labstack/echo/v4"
)

func InitV2Router() http.Handler {
	swagger, err := codegen.GetSwagger()
	if err != nil {
		panic(err)
	}

	localStorage := v2.NewLocalStorage()

	e := echo.New()

	e.Use(middleware.OapiRequestValidator(swagger))

	codegen.RegisterHandlersWithBaseURL(e, localStorage, swagger.Servers[0].URL)

	return e
}

func InitV2DocRouter(docHTML string, docYAML string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/doc/v2/local_storage" {
			if _, err := w.Write([]byte(docHTML)); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		if r.URL.Path == "/doc/v2/local_storage/openapi.yaml" {
			if _, err := w.Write([]byte(docYAML)); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	})
}
