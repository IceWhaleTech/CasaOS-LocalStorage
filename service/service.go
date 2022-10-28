package service

import (
	external "github.com/IceWhaleTech/CasaOS-Common/service/v1"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/config"
	v2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2/wrapper"
	"github.com/IceWhaleTech/CasaOS/common"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
)

var Cache *cache.Cache

var MyService Repository

type Repository interface {
	Disk() DiskService
	USB() USBService
	LocalStorage() *v2.LocalStorageService
	Gateway() external.ManagementService
	Notify() common.NotifyService
	Shares() common.ShareService
}

func NewService(db *gorm.DB) Repository {
	gatewayManagement, err := external.NewManagementService(config.CommonInfo.RuntimePath)
	if err != nil {
		panic(err)
	}

	notifyService, err := common.NewNotifyService(config.CommonInfo.RuntimePath)
	if err != nil {
		panic(err)
	}

	sharesService, err := common.NewShareService(config.CommonInfo.RuntimePath)
	if err != nil {
		panic(err)
	}

	return &store{
		usb:          NewUSBService(),
		disk:         NewDiskService(db),
		localStorage: v2.NewLocalStorageService(db, wrapper.NewMountInfo()),
		gateway:      gatewayManagement,
		notify:       notifyService,
		shares:       sharesService,
	}
}

type store struct {
	usb          USBService
	disk         DiskService
	localStorage *v2.LocalStorageService
	gateway      external.ManagementService
	notify       common.NotifyService
	shares       common.ShareService
}

func (c *store) Gateway() external.ManagementService {
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

func (c *store) Notify() common.NotifyService {
	return c.notify
}

func (c *store) Shares() common.ShareService {
	return c.shares
}
