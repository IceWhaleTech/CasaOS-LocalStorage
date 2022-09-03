package config

import (
	"github.com/IceWhaleTech/CasaOS-LocalStorage/model"
	"gopkg.in/ini.v1"
)

var (
	AppInfo          = &model.APPModel{}
	ServerInfo       = &model.ServerModel{}
	SystemConfigInfo = &model.SystemConfig{}
)

var Cfg *ini.File
