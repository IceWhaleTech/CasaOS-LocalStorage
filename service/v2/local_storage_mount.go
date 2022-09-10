package v2

import (
	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2/adapter"
	"github.com/moby/sys/mountinfo"
)

func (s *LocalStorageService) GetMounts() ([]codegen.Mount, error) {
	mounts, err := mountinfo.GetMounts(nil)
	if err != nil {
		return nil, err
	}

	mountList := make([]codegen.Mount, len(mounts))
	for i, mount := range mounts {
		mountList[i] = adapter.GetMount(mount)
	}

	return mountList, nil
}
