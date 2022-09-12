package wrapper

import "github.com/moby/sys/mountinfo"

// This interface is used to mock the mountinfo package in unit tests.
type MountInfoWrapper interface {
	GetMounts(f mountinfo.FilterFunc) ([]*mountinfo.Info, error)
}

type MountInfo struct{}

func NewMountInfo() MountInfoWrapper {
	return &MountInfo{}
}

func (m *MountInfo) GetMounts(f mountinfo.FilterFunc) ([]*mountinfo.Info, error) {
	return mountinfo.GetMounts(f)
}
