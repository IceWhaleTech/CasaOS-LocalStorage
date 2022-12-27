package service

import (
	"github.com/IceWhaleTech/CasaOS-Common/external"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen/message_bus"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/config"
	v2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2/wrapper"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
)

var Cache *cache.Cache

var MyService Services

type Services interface {
	Disk() DiskService
	USB() USBService
	LocalStorage() *v2.LocalStorageService
	Gateway() external.ManagementService
	Notify() external.NotifyService
	Shares() external.ShareService
	MessageBus() *message_bus.ClientWithResponses
}

func NewService(db *gorm.DB) Services {
	gatewayManagement, err := external.NewManagementService(config.CommonInfo.RuntimePath)
	if err != nil {
		panic(err)
	}

	notifyService := external.NewNotifyService(config.CommonInfo.RuntimePath)
	sharesService := external.NewShareService(config.CommonInfo.RuntimePath)

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
	notify       external.NotifyService
	shares       external.ShareService
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

func (c *store) Notify() external.NotifyService {
	return c.notify
}

func (c *store) Shares() external.ShareService {
	return c.shares
}

func (c *store) MessageBus() *message_bus.ClientWithResponses {
	client, _ := message_bus.NewClientWithResponses("", func(c *message_bus.Client) error {
		// error will never be returned, as we always want to return a client, even with wrong address,
		// in order to avoid panic.
		//
		// If we don't avoid panic, message bus becomes a hard dependency, which is not what we want.

		messageBusAddress, err := external.GetMessageBusAddress(config.CommonInfo.RuntimePath)
		if err != nil {
			c.Server = "message bus address not found"
			return nil
		}

		c.Server = messageBusAddress
		return nil
	})

	return client
}
