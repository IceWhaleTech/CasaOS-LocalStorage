package v2

import (
	"errors"
	"net/http"
	"syscall"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	"github.com/labstack/echo/v4"
)

type MountError interface {
	Error() string
	Cause() error
	Unwrap() error
}

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
	var request codegen.Mount
	if err := ctx.Bind(&request); err != nil {
		message := err.Error()
		response := codegen.BaseResponse{
			Message: &message,
		}
		return ctx.JSON(http.StatusBadRequest, response)
	}

	mount, err := s.service.Mount(*request.Source, *request.Mountpoint, *request.FSType, *request.Options)
	if err != nil {

		var mountError MountError
		var internalError syscall.Errno
		if errors.As(err, &mountError) && errors.As(mountError.Unwrap(), &internalError) && internalError == syscall.EPERM {
			message := err.Error()
			response := codegen.BaseResponse{
				Message: &message,
			}
			return ctx.JSON(http.StatusForbidden, response)
		}

		message := err.Error()
		response := codegen.BaseResponse{
			Message: &message,
		}
		return ctx.JSON(http.StatusInternalServerError, response)
	}

	if request.Persist != nil && *request.Persist {
		// TODO - persist mount to fstab

		message := "Persisting mounts to fstab is not yet implemented"
		return ctx.JSON(http.StatusNotImplemented, codegen.BaseResponse{
			Message: &message,
		})
	}

	return ctx.JSON(http.StatusOK, codegen.MountResponseOK{
		Data: mount,
	})
}
