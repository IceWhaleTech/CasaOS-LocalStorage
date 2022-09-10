package v2

import (
	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	v2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2"
)

type LocalStorage struct {
	service *v2.LocalStorageService
}

func NewLocalStorage() codegen.ServerInterface {
	return &LocalStorage{
		service: v2.NewLocalStorageService(),
	}
}
