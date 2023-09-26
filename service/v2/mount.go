package v2

import (
	"errors"
	"strconv"

	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/mount"

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
		results[i] = *fs.ExtendAll(MountAdapter(mountInfo))
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
	logger.Info("checking if mount point exist", zap.String("mount point", m.MountPoint))
	if empty, err := file.IsDirEmpty(m.MountPoint); err != nil {
		logger.Error("error when trying to check if mount point is empty", zap.Error(err), zap.Any("mount", m))
		return nil, err
	} else if !empty {
		logger.Error("mount point is not empty", zap.Any("mount", m))
		return nil, ErrMountPointIsNotEmpty
	}

	if err := mount.Mount(*m.Source, m.MountPoint, m.Fstype, m.Options); err != nil {
		logger.Error("error when trying to mount", zap.Error(err), zap.Any("mount", m))
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
		logger.Error("More than one mount with same mount point and fstype found", zap.Any("mounts", results))
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
		logger.Error("Error when trying to get mounted volume", zap.Error(err), zap.String("mount point", mountpoint))
		return err
	}

	if len(results) == 0 {
		logger.Info("not mounted", zap.String("mount point", mountpoint))
		return ErrNotMounted
	}

	if err := mount.UmountByMountPoint(mountpoint); err != nil {
		logger.Error("error when trying to umount by mount point", zap.Error(err), zap.String("mount point", mountpoint))
		return err
	}

	return nil
}

func MountAdapter(m *mountinfo.Info) codegen.Mount {
	return codegen.Mount{
		MountPoint: m.Mountpoint,

		Id:      &m.ID,
		Options: &m.Options,
		Source:  &m.Source,
		Fstype:  &m.FSType,
	}
}
