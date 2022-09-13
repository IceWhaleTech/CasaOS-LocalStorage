package v2

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2/adapter"
	"github.com/moby/sys/mountinfo"
	"go.uber.org/zap"
)

var ErrAlreadyMounted = errors.New("volume is already mounted")

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
		logger.Error("Error when trying to get mounted volumes: %v", zap.Any("error", err))
		return nil, err
	}

	results := make([]codegen.Mount, len(mounts))

	for i, mountInfo := range mounts {
		results[i] = adapter.GetMount(mountInfo)
	}

	return results, nil
}

func (s *LocalStorageService) Mount(source, mountpoint, fstype, options string) (*codegen.Mount, error) {
	// check if mountpoint is already mounted
	results, err := s.GetMounts(codegen.GetMountsParams{
		MountPoint: &mountpoint,
		Type:       &fstype,
	})
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		return &results[0], ErrAlreadyMounted
	}

	cmd := exec.Command("mount", "-t", fstype, source, mountpoint, "-o", options)
	if _, err := cmd.Output(); err != nil {
		return nil, err
	}

	results, err = s.GetMounts(codegen.GetMountsParams{
		MountPoint: &mountpoint,
		Type:       &fstype,
	})
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	if len(results) > 1 {
		fmt.Printf("Mount source `%s` of type `%s` to mount point `%s` with options `%s`, but got %d results", source, fstype, mountpoint, options, len(results))
	}

	return &results[0], nil
}
