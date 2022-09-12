package v2

import (
	"net/http"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	"github.com/labstack/echo/v4"
)

func (s *LocalStorage) GetMounts(ctx echo.Context, params codegen.GetMountsParams) error {
	mounts, err := s.service.GetMounts(params)
	if err != nil {
		message := err.Error()
		response := codegen.BaseResponse{
			Message: &message,
		}
		return ctx.JSON(http.StatusInternalServerError, response)
	}

	return ctx.JSON(http.StatusOK, codegen.GetMountsResponseOK{
		Data: &mounts,
	})
}
