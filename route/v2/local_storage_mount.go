package v2

import (
	"net/http"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	"github.com/labstack/echo/v4"
)

func (s *LocalStorage) GetMounts(ctx echo.Context) error {
	mounts, err := s.service.GetMounts()
	if err != nil {
		message := err.Error()
		response := codegen.BaseResponse{
			Message: &message,
		}
		return ctx.JSON(http.StatusInternalServerError, response)
	}

	response := codegen.GetMountsResponseOK{
		Data: &mounts,
	}

	return ctx.JSON(http.StatusOK, response)
}
