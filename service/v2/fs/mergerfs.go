package fs

import (
	"strings"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
)

const (
	MergerFS               = "mergerfs"
	MergerFSFullName       = "fuse.mergerfs"
	MergerFSDefaultOptions = "category.create=mfs,moveonenospc=true,minfreespace=1M"

	MergerFSExtendedKeySource = "mergerfs.src" // corresponding value could be for example: /var/lib/casaos/files

)

type mergerFS struct{}

func init() {
	if ExtensionMap == nil {
		ExtensionMap = make(map[string]Extension)
	}

	// register itself to ExtensionMap
	ExtensionMap[MergerFS] = &mergerFS{}
}

func (f *mergerFS) GetFSType() string {
	return MergerFS
}

func (f *mergerFS) GetFSTypeFull() string {
	return MergerFSFullName
}

func (f *mergerFS) PreMount(m codegen.Mount) *codegen.Mount {
	if *m.Fstype != MergerFSFullName && *m.Fstype != MergerFS {
		return &m
	}

	mNew := m

	mNew = updateOptions(mNew, MergerFSDefaultOptions)
	mNew = updateFSType(mNew)
	mNew = updateFSName(mNew)

	return &mNew
}

func (f *mergerFS) PostMount(m codegen.Mount) *codegen.Mount {
	return &m
}

func (f *mergerFS) Extend(m codegen.Mount) *codegen.Mount {
	if *m.Fstype != MergerFSFullName && *m.Fstype != MergerFS {
		return &m
	}

	mNew := m

	mNew = updateExtended(mNew)

	return &mNew
}

func updateOptions(m codegen.Mount, options string) codegen.Mount {
	if m.Options == nil || *m.Options == "" {
		m.Options = &options
	}

	return m
}

func updateFSType(m codegen.Mount) codegen.Mount {
	if *m.Fstype == MergerFS {
		f := MergerFSFullName
		m.Fstype = &f
	}

	return m
}

func updateFSName(m codegen.Mount) codegen.Mount {
	options := ""

	if m.Options != nil {
		if strings.Contains(strings.ToLower(*m.Options), "fsname=") {
			return m
		}
		options = *m.Options
	}

	optionNew := strings.TrimLeft(options+",fsname="+*m.Source, ",")

	m.Options = &optionNew

	return m
}

func updateExtended(m codegen.Mount) codegen.Mount {
	if m.Extended == nil {
		extended := make(map[string]string)
		m.Extended = &extended
	}

	(*m.Extended)[MergerFSExtendedKeySource] = *m.Source

	return m
}
