package fs

import (
	"strings"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
)

const (
	fsType     = "mergerfs"
	fsTypeFull = "fuse.mergerfs"
)

type mergerFS struct{}

func init() {
	if ExtensionMap == nil {
		ExtensionMap = make(map[string]Extension)
	}

	ExtensionMap[fsType] = &mergerFS{}
}

func (f *mergerFS) PreMount(m codegen.Mount) *codegen.Mount {
	if *m.FSType != fsTypeFull && *m.FSType != fsType {
		return &m
	}

	mNew := m

	mNew = updateFSType(mNew)
	mNew = updateFSName(mNew)

	return &mNew
}

func (f *mergerFS) PostMount(m codegen.Mount) *codegen.Mount {
	return &m
}

func (f *mergerFS) Extend(m codegen.Mount) *codegen.Mount {
	if *m.FSType != fsTypeFull && *m.FSType != fsType {
		return &m
	}

	mNew := m

	mNew = updateExtended(mNew)

	return &mNew
}

func updateFSType(m codegen.Mount) codegen.Mount {
	if *m.FSType == fsType {
		f := fsTypeFull
		m.FSType = &f
	}

	return m
}

func updateFSName(m codegen.Mount) codegen.Mount {
	if strings.Contains(strings.ToLower(*m.Options), "fsname=") {
		return m
	}

	optionNew := strings.TrimLeft(*m.Options+",fsname="+*m.Source, ",")

	m.Options = &optionNew

	return m
}

func updateExtended(m codegen.Mount) codegen.Mount {
	if m.Extended == nil {
		m.Extended = &codegen.Mount_Extended{}
	}

	if m.Extended.AdditionalProperties == nil {
		m.Extended.AdditionalProperties = make(map[string]string)
	}

	m.Extended.AdditionalProperties["mergerfs.srcmounts"] = *m.Source

	return m
}
