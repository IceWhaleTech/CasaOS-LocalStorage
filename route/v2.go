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
