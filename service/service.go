package service

import (
	gateway "github.com/IceWhaleTech/CasaOS-Gateway/common"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
)

var Cache *cache.Cache

var MyService Repository

type Repository interface {
	Disk() DiskService
	Gateway() gateway.ManagementService
}

func NewService(db *gorm.DB, runtimePath string) Repository {
	gatewayManagement, err := gateway.NewManagementService(runtimePath)
	if err != nil {
		panic(err)
	}

	return &store{
		gateway: gatewayManagement,
		disk:    NewDiskService(db),
	}
}

type store struct {
	db      *gorm.DB
	disk    DiskService
	gateway gateway.ManagementService
}

func (c *store) Gateway() gateway.ManagementService {
	return c.gateway
}

func (c *store) Disk() DiskService {
	return c.disk
}
