package adapter

import (
	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	"github.com/moby/sys/mountinfo"
)

func GetMount(m *mountinfo.Info) codegen.Mount {
	return codegen.Mount{
		MountPoint: m.Mountpoint,

		Id:      &m.ID,
		Options: &m.Options,
		Source:  &m.Source,
		Fstype:  &m.FSType,
	}
}
