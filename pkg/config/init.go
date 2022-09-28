package config

import (
	"log"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/model"
	"gopkg.in/ini.v1"
)

var (
	CommonInfo = &model.CommonModel{
		RuntimePath: "/var/run/casaos",
	}

	AppInfo = &model.APPModel{
		DBPath:      "/var/lib/casaos",
		LogPath:     "/var/log/casaos",
		LogSaveName: "local-storage",
		LogFileExt:  "log",
	}

	ServerInfo = &model.ServerModel{
		USBAutoMount:   "False",
		EnableMergerFS: "False",
	}
)

var (
	Cfg            *ini.File
	ConfigFilePath string
)

func InitSetup(config string) {
	ConfigFilePath = LocalStorageConfigFilePath
	if len(config) > 0 {
		ConfigFilePath = config
	}

	var err error

	Cfg, err = ini.Load(ConfigFilePath)
	if err != nil {
		panic(err)
	}

	mapTo("common", CommonInfo)
	mapTo("app", AppInfo)
	mapTo("server", ServerInfo)
}

// 映射
func mapTo(section string, v interface{}) {
	err := Cfg.Section(section).MapTo(v)
	if err != nil {
		log.Fatalf("Cfg.MapTo %s err: %v", section, err)
	}
}
