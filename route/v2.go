package route

import (
	"net/http"

	"github.com/IceWhaleTech/CasaOS-Common/utils/jwt"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service"
	openapi "github.com/IceWhaleTech/CasaOS-LocalStorage/target/codegen"
)

func InitV2Router() http.Handler {
	v2Controller := openapi.NewDefaultApiController(service.NewStorageService())

	router := openapi.NewRouter(v2Controller)

	return jwt.ExceptLocalhost2(router)
}

func InitV2DocRouter(docHTML string, docYAML string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/doc/v1/storage" {
			if _, err := w.Write([]byte(docHTML)); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		if r.URL.Path == "/doc/v1/storage/openapi.yaml" {
			if _, err := w.Write([]byte(docYAML)); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	})
}
