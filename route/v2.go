package route

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	v2 "github.com/IceWhaleTech/CasaOS-LocalStorage/route/v2"
	"github.com/deepmap/oapi-codegen/pkg/middleware"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
)

var (
	_swagger *openapi3.T

	V2APIPath string
	V2DocPath string
)

func init() {
	swagger, err := codegen.GetSwagger()
	if err != nil {
		panic(err)
	}

	_swagger = swagger

	u, err := url.Parse(_swagger.Servers[0].URL)
	if err != nil {
		panic(err)
	}

	V2APIPath = strings.TrimRight(u.Path, "/")
	V2DocPath = "/doc" + V2APIPath
}

func InitV2Router() http.Handler {
	localStorage := v2.NewLocalStorage()

	e := echo.New()

	e.Use(middleware.OapiRequestValidator(_swagger))

	// TODO - add JWT2 here

	codegen.RegisterHandlersWithBaseURL(e, localStorage, V2APIPath)

	return e
}

func InitV2DocRouter(docHTML string, docYAML string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == V2DocPath {
			if _, err := w.Write([]byte(docHTML)); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		if r.URL.Path == V2DocPath+"/openapi.yaml" {
			if _, err := w.Write([]byte(docYAML)); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	})
}
