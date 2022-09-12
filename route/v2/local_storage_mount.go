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

func (s *LocalStorage) Mount(ctx echo.Context) error {
	var mountRequest codegen.MountRequest
	if err := ctx.Bind(&mountRequest); err != nil {
		message := err.Error()
		response := codegen.BaseResponse{
			Message: &message,
		}
		return ctx.JSON(http.StatusBadRequest, response)
	}

	mount, err := s.service.Mount(*mountRequest.Mount.Source, *mountRequest.Mount.Mountpoint, *mountRequest.Mount.FSType, *mountRequest.Mount.Options)
	if err != nil {
		message := err.Error()
		response := codegen.BaseResponse{
			Message: &message,
		}
		return ctx.JSON(http.StatusInternalServerError, response)
	}

	return ctx.JSON(http.StatusOK, codegen.MountResponseOK{
		Data: mount,
	})
}
