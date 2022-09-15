package service

import (
	gateway "github.com/IceWhaleTech/CasaOS-Gateway/common"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/config"
	v2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2/wrapper"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
)

var Cache *cache.Cache

var MyService Repository

type Repository interface {
	Disk() DiskService
	USB() USBService
	LocalStorage() *v2.LocalStorageService
	Gateway() gateway.ManagementService
}

func NewService(db *gorm.DB) Repository {
	gatewayManagement, err := gateway.NewManagementService(config.CommonInfo.RuntimePath)
	if err != nil {
		panic(err)
	}

	return &store{
		usb:          NewUSBService(),
		disk:         NewDiskService(db),
		localStorage: v2.NewLocalStorageService(wrapper.NewMountInfo()),
		gateway:      gatewayManagement,
	}
}

type store struct {
	usb          USBService
	disk         DiskService
	localStorage *v2.LocalStorageService
	gateway      gateway.ManagementService
}

func (c *store) Gateway() gateway.ManagementService {
	return c.gateway
}

func (c *store) USB() USBService {
	return c.usb
}

func (c *store) Disk() DiskService {
	return c.disk
}

func (c *store) LocalStorage() *v2.LocalStorageService {
	return c.localStorage
}
