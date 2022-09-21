package v2

import (
	"errors"
	"os/exec"
	"strconv"

	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2/adapter"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2/fs"
	"github.com/moby/sys/mountinfo"
	"go.uber.org/zap"
)

var (
	ErrNotMounted           = errors.New("not mounted")
	ErrAlreadyMounted       = errors.New("volume is already mounted")
	ErrMountPointIsNotEmpty = errors.New("mountpoint is not empty")
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
		logger.Error("Error when trying to get mounted volume(s)", zap.Error(err))
		return nil, err
	}

	results := make([]codegen.Mount, len(mounts))

	for i, mountInfo := range mounts {
		results[i] = *fs.ExtendAll(adapter.GetMount(mountInfo))
	}

	return results, nil
}

func (s *LocalStorageService) Mount(m codegen.Mount) (*codegen.Mount, error) {
	m = *fs.PreMountAll(m)

	// check if mountpoint is already mounted
	results, err := s.GetMounts(codegen.GetMountsParams{
		MountPoint: &m.MountPoint,
		Type:       m.Fstype,
	})
	if err != nil {
		logger.Error("Error when trying to get mounted volume", zap.Error(err), zap.Any("mount", m))
		return nil, err
	}

	if len(results) > 0 {
		logger.Info("Volume is already mounted", zap.Any("mount", results[0]))
		return &results[0], ErrAlreadyMounted
	}

	// check if mountpoint is empty
	if empty, err := file.IsDirEmpty(m.MountPoint); err != nil {
		logger.Error("Error when trying to check if mountpoint is empty", zap.Error(err), zap.Any("mount", m))
		return nil, err
	} else if !empty {
		logger.Error("MountPoint is not empty", zap.Any("mount", m))
		return nil, ErrMountPointIsNotEmpty
	}

	cmd := exec.Command("mount", "-t", *m.Fstype, *m.Source, m.MountPoint, "-o", *m.Options) // #nosec
	logger.Info("Executing command", zap.Any("command", cmd.String()))
	if buf, err := cmd.CombinedOutput(); err != nil {
		logger.Error(string(buf), zap.Error(err), zap.Any("mount", m))
		return nil, err
	}

	results, err = s.GetMounts(codegen.GetMountsParams{
		MountPoint: &m.MountPoint,
		Type:       m.Fstype,
	})
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	if len(results) > 1 {
	}

	results[0] = *fs.PostMountAll(results[0])

	return &results[0], nil
}

func (s *LocalStorageService) Umount(mountpoint string) error {
	// check if mountpoint is already mounted
	results, err := s.GetMounts(codegen.GetMountsParams{
		MountPoint: &mountpoint,
	})
	if err != nil {
		logger.Error("Error when trying to get mounted volume", zap.Error(err), zap.Any("mountpoint", mountpoint))
		return err
	}

	if len(results) == 0 {
		logger.Info("not mounted", zap.Any("mountpoint", mountpoint))
		return ErrNotMounted
	}

	cmd := exec.Command("umount", mountpoint) // #nosec
	logger.Info("Executing command", zap.Any("command", cmd.String()))
	if buf, err := cmd.CombinedOutput(); err != nil {
		logger.Error(string(buf), zap.Error(err), zap.Any("mountpoint", mountpoint))
		return err
	}
	return nil
}
