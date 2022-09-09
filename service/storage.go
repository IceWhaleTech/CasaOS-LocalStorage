package service

import (
	"context"

	openapi "github.com/IceWhaleTech/CasaOS-LocalStorage/target/codegen"
)

type storageService struct{}

func (s *storageService) StorageGet(c context.Context) (openapi.ImplResponse, error) {
	return openapi.ImplResponse{
		Code: 200,
		Body: openapi.StorageDevice{},
	}, nil
}

func (s *storageService) StoragePost(c context.Context, storageDevice openapi.StorageDevice) (openapi.ImplResponse, error) {
	return openapi.ImplResponse{}, nil
}

func NewStorageService() openapi.DefaultApiServicer {
	return &storageService{}
}
