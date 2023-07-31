package config

import (
	"fmt"
	"log"
	"os"

	"github.com/IceWhaleTech/CasaOS-Common/utils/constants"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/model"
	"gopkg.in/ini.v1"
)

var (
	CommonInfo = &model.CommonModel{
		RuntimePath: constants.DefaultRuntimePath,
	}

	AppInfo = &model.APPModel{
		DBPath:      constants.DefaultDataPath,
		LogPath:     constants.DefaultLogPath,
		LogSaveName: "local-storage",
		LogFileExt:  "log",
		ShellPath:   "/usr/share/casaos/shell",
	}

	ServerInfo = &model.ServerModel{
		USBAutoMount:   "True",
		EnableMergerFS: "False",
	}
)

var (
	Cfg            *ini.File
	ConfigFilePath string
)

func InitSetup(config string, sample string) {
	ConfigFilePath = LocalStorageConfigFilePath
	if len(config) > 0 {
		ConfigFilePath = config
	}

	// create default config file if not exist
	if _, err := os.Stat(ConfigFilePath); os.IsNotExist(err) {
		fmt.Println("config file not exist, create it")
		// create config file
		file, err := os.Create(ConfigFilePath)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		// write default config
		_, err = file.WriteString(sample)
		if err != nil {
			panic(err)
		}
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

func SaveSetup(config string) {
	reflectFrom("common", CommonInfo)
	reflectFrom("app", AppInfo)
	reflectFrom("server", ServerInfo)

	configFilePath := LocalStorageConfigFilePath
	if len(config) > 0 {
		configFilePath = config
	}

	if err := Cfg.SaveTo(configFilePath); err != nil {
		log.Printf("error when saving to %s", configFilePath)
		panic(err)
	}
}

// 映射
func mapTo(section string, v interface{}) {
	err := Cfg.Section(section).MapTo(v)
	if err != nil {
		log.Fatalf("Cfg.MapTo %s err: %v", section, err)
	}
}

func reflectFrom(section string, v interface{}) {
	err := Cfg.Section(section).ReflectFrom(v)
	if err != nil {
		log.Fatalf("Cfg.ReflectFrom %s err: %v", section, err)
	}
}
