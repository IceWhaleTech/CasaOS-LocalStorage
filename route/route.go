package route

import (
	"os"

	"github.com/IceWhaleTech/CasaOS-Common/middleware"
	"github.com/IceWhaleTech/CasaOS-Common/utils/jwt"
	v1 "github.com/IceWhaleTech/CasaOS-LocalStorage/route/v1"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	r := gin.Default()
	r.Use(middleware.Cors())
	r.Use(middleware.WriteLog())
	r.Use(gzip.Gzip(gzip.DefaultCompression))

	// check if environment variable is set
	if ginMode, success := os.LookupEnv("GIN_MODE"); success {
		gin.SetMode(ginMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	v1Group := r.Group("/v1")

	v1Group.Use(jwt.JWT())

	{
		v1DisksGroup := v1Group.Group("/disks")
		v1DisksGroup.Use()
		{
			// v1DiskGroup.GET("/check", v1.GetDiskCheck) //delete
			// v1DisksGroup.GET("", v1.GetDiskInfo)

			// v1DisksGroup.POST("", v1.PostMountDisk)
			v1DisksGroup.GET("", v1.GetDiskList)
			v1DisksGroup.GET("/usb", v1.GetDisksUSBList)
			v1DisksGroup.DELETE("/usb", v1.DeleteDiskUSB)
			v1DisksGroup.DELETE("", v1.DeleteDisksUmount)
			// //format storage
			// v1DiskGroup.POST("/format", v1.PostDiskFormat)

			// //mount SATA disk
			// v1DiskGroup.POST("/mount", v1.PostMountDisk)

			// //umount sata disk
			// v1DiskGroup.POST("/umount", v1.PostDiskUmount)

			// v1DiskGroup.GET("/type", v1.FormatDiskType)//delete

			v1DisksGroup.DELETE("/part", v1.RemovePartition) // disk/delpart
		}

		v1StorageGroup := v1Group.Group("/storage")
		v1StorageGroup.Use()
		{
			v1StorageGroup.POST("", v1.PostDiskAddPartition)

			v1StorageGroup.PUT("", v1.PostDiskFormat)

			v1StorageGroup.DELETE("", v1.PostDiskUmount)
			v1StorageGroup.GET("", v1.GetStorageList)
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
