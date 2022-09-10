package adapter

import (
	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	"github.com/moby/sys/mountinfo"
)

func GetMount(m *mountinfo.Info) codegen.Mount {
	return codegen.Mount{
		Id:         &m.ID,
		Mountpoint: &m.Mountpoint,
		Options:    &m.Options,
		Source:     &m.Source,
		Type:       &m.FSType,
	}
}
