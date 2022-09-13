package v2

import (
	"errors"
	"net/http"
	"syscall"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	v2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2"

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
		return ctx.JSON(http.StatusBadRequest, codegen.MountResponseBadRequest{Message: &message})
	}

	mount, err := s.service.Mount(request)
	if err != nil {
		var mountError MountError
		var internalError syscall.Errno

		message := err.Error()

		if errors.As(err, &mountError) && errors.As(mountError.Unwrap(), &internalError) && internalError == syscall.EPERM {
			return ctx.JSON(http.StatusForbidden, codegen.MountResponseForbidden{Message: &message})
		}

		if errors.Is(err, v2.ErrAlreadyMounted) || errors.Is(err, v2.ErrMountpointIsNotEmpty) {
			return ctx.JSON(http.StatusConflict, codegen.MountResponseConflict{Message: &message})
		}

		return ctx.JSON(http.StatusInternalServerError, codegen.BaseResponse{Message: &message})
	}

	if request.Persist != nil && *request.Persist {
		// TODO - persist mount to fstab

		message := "Persisting mounts to fstab is not yet implemented"
		return ctx.JSON(http.StatusNotImplemented, codegen.BaseResponse{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.MountResponseOK{Data: mount})
}
