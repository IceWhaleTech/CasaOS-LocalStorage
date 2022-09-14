package v2

import (
	"github.com/IceWhaleTech/CasaOS-LocalStorage/common"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2/wrapper"
)

type LocalStorageService struct {
	_mountinfo wrapper.MountInfoWrapper
	_fstab     common.FStab
}

func NewLocalStorageService(mountinfo wrapper.MountInfoWrapper) *LocalStorageService {
	return &LocalStorageService{
		_mountinfo: mountinfo,
		_fstab:     *common.GetFSTab(),
	}
}
