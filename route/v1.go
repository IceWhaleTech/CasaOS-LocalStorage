package route

import (
	"os"

	"github.com/IceWhaleTech/CasaOS-Common/middleware"
	"github.com/IceWhaleTech/CasaOS-Common/utils/jwt"
	v1 "github.com/IceWhaleTech/CasaOS-LocalStorage/route/v1"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func InitV1Router() *gin.Engine {
	// check if environment variable is set
	ginMode, success := os.LookupEnv(gin.EnvGinMode)
	if !success {
		ginMode = gin.ReleaseMode
	}
	gin.SetMode(ginMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Cors())
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	if ginMode != gin.ReleaseMode {
		r.Use(middleware.WriteLog())
	}
	r.GET("/v1/recover/:type", v1.GetRecoverStorage)
	v1Group := r.Group("/v1")

	v1Group.Use(jwt.JWT())

	{
		v1DisksGroup := v1Group.Group("/disks")
		v1DisksGroup.Use()
		{

			v1DisksGroup.GET("", v1.GetDiskList)
			v1DisksGroup.GET("/usb", v1.GetDisksUSBList)
			v1DisksGroup.DELETE("/usb", v1.DeleteDiskUSB)
			v1DisksGroup.DELETE("", v1.DeleteDisksUmount)
		}

		v1StorageGroup := v1Group.Group("/storage")
		v1StorageGroup.Use()
		{
			v1StorageGroup.POST("", v1.PostAddStorage)

			v1StorageGroup.PUT("", v1.PutFormatStorage)

			v1StorageGroup.DELETE("", v1.DeleteStorage)
			v1StorageGroup.GET("", v1.GetStorageList)
		}
		v1CloudGroup := v1Group.Group("/cloud")
		v1CloudGroup.Use()
		{
			v1CloudGroup.GET("", v1.ListStorages)
			v1CloudGroup.DELETE("", v1.UmountStorage)
		}
		v1DriverGroup := v1Group.Group("/driver")
		v1DriverGroup.Use()
		{
			v1DriverGroup.GET("", v1.ListDriverInfo)
		}
		v1USBGroup := v1Group.Group("/usb")
		v1USBGroup.Use()
		{
			v1USBGroup.PUT("/usb-auto-mount", v1.PutSystemUSBAutoMount) ///sys/usb/:status
			v1USBGroup.GET("/usb-auto-mount", v1.GetSystemUSBAutoMount) ///sys/usb/status
		}
	}

	return r
}
