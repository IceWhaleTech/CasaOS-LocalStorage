package v2

import (
	"strconv"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2/adapter"
	"github.com/moby/sys/mountinfo"
)

func (s *LocalStorageService) GetMounts(params codegen.GetMountsParams) ([]codegen.Mount, error) {
	mounts, err := s._mountinfo.GetMounts(func(i *mountinfo.Info) (skip bool, stop bool) {
		if params.Id != nil {
			if strconv.Itoa(i.ID) != *params.Id {
				return true, false
			}
		}
		if params.MountPoint != nil {
			if i.Mountpoint != *params.MountPoint {
				return true, false
			}
		}
		if params.Type != nil {
			if i.FSType != *params.Type {
				return true, false
			}
		}
		if params.Source != nil {
			if i.Source != *params.Source {
				return true, false
			}
		}
		return false, false
	})
	if err != nil {
		return nil, err
	}

	results := make([]codegen.Mount, len(mounts))

	for i, mountInfo := range mounts {
		results[i] = adapter.GetMount(mountInfo)
	}

	return results, nil
}
