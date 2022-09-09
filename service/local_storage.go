package service

import (
	openapi "github.com/IceWhaleTech/CasaOS-LocalStorage/target/codegen"
)

type localStorageService struct{}

func NewStorageService() openapi.DefaultApiServicer {
	return &localStorageService{}
}
