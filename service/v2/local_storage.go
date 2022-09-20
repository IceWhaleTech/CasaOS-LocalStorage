package v2

import (
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/fstab"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2/wrapper"
	"gorm.io/gorm"
)

type LocalStorageService struct {
	_mountinfo wrapper.MountInfoWrapper
	_fstab     fstab.FStab
	_db        *gorm.DB
}

func NewLocalStorageService(db *gorm.DB, mountinfo wrapper.MountInfoWrapper) *LocalStorageService {
	return &LocalStorageService{
		_mountinfo: mountinfo,
		_fstab:     *fstab.New(),
	}
}
