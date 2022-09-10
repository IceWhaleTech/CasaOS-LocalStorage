package v2

import "github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"

type LocalStorage struct{}

func NewLocalStorage() codegen.ServerInterface {
	return &LocalStorage{}
}
