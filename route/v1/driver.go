package v1

import (
	"github.com/IceWhaleTech/CasaOS-Common/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/common_err"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/internal/op"
	"github.com/labstack/echo/v4"
)

func ListDriverInfo(ctx echo.Context) error {
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: op.GetDriverInfoMap()})
}
