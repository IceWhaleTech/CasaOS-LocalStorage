package v2

import (
	"errors"
	"net/http"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service"
	v2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2"

	"github.com/labstack/echo/v4"
)

type MountError interface {
	Error() string
	Cause() error
	Unwrap() error
}

func (s *LocalStorage) GetMounts(ctx echo.Context, params codegen.GetMountsParams) error {
	mounts, err := service.MyService.LocalStorage().GetMounts(params)
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
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	mount, err := service.MyService.LocalStorage().Mount(request)
	if err != nil {
		message := err.Error()

		if errors.Is(err, v2.ErrAlreadyMounted) || errors.Is(err, v2.ErrMountPointIsNotEmpty) {
			return ctx.JSON(http.StatusConflict, codegen.ResponseConflict{Message: &message})
		}

		return ctx.JSON(http.StatusInternalServerError, codegen.BaseResponse{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.AddMountResponseOK{Data: mount})
}

func (s *LocalStorage) Umount(ctx echo.Context, params codegen.UmountParams) error {
	if err := service.MyService.LocalStorage().Umount(params.MountPoint); err != nil {
		message := err.Error()

		if errors.Is(err, v2.ErrNotMounted) {
			return ctx.JSON(http.StatusNotFound, codegen.ResponseNotFound{Message: &message})
		}

		return ctx.JSON(http.StatusInternalServerError, codegen.BaseResponse{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.UmountResponseOK{})
}

func (s *LocalStorage) UpdateMount(ctx echo.Context, params codegen.UpdateMountParams) error {
	var request codegen.Mount
	if err := ctx.Bind(&request); err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	if err := service.MyService.LocalStorage().Umount(params.MountPoint); err != nil {
		message := err.Error()

		if !errors.Is(err, v2.ErrNotMounted) {
			return ctx.JSON(http.StatusInternalServerError, codegen.BaseResponse{Message: &message})
		}
	}

	mount, err := service.MyService.LocalStorage().Mount(request)
	if err != nil {
		message := err.Error()

		if errors.Is(err, v2.ErrAlreadyMounted) || errors.Is(err, v2.ErrMountPointIsNotEmpty) {
			return ctx.JSON(http.StatusConflict, codegen.ResponseConflict{Message: &message})
		}

		return ctx.JSON(http.StatusInternalServerError, codegen.BaseResponse{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.AddMountResponseOK{Data: mount})
}
