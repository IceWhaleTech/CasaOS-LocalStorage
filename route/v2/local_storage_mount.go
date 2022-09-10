package v2

import (
	"net/http"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	"github.com/labstack/echo/v4"
)

func (s *LocalStorage) GetMount(ctx echo.Context) error {
	m := codegen.Mount{}
	return ctx.JSON(http.StatusOK, m)
}
