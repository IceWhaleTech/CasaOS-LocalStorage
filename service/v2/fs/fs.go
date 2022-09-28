package fs

import "github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"

type Extension interface {
	GetFSType() string
	GetFSTypeFull() string

	PreMount(m codegen.Mount) *codegen.Mount
	PostMount(m codegen.Mount) *codegen.Mount
	Extend(m codegen.Mount) *codegen.Mount
}

var ExtensionMap map[string]Extension

func init() {
	if ExtensionMap == nil {
		ExtensionMap = make(map[string]Extension)
	}
}

func PreMountAll(m codegen.Mount) *codegen.Mount {
	for _, ext := range ExtensionMap {
		m = *ext.PreMount(m)
	}

	return &m
}

func PostMountAll(m codegen.Mount) *codegen.Mount {
	for _, ext := range ExtensionMap {
		m = *ext.PostMount(m)
	}

	return &m
}

func ExtendAll(m codegen.Mount) *codegen.Mount {
	for _, ext := range ExtensionMap {
		m = *ext.Extend(m)
	}

	return &m
}
