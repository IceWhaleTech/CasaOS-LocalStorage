package model

type APPModel struct {
	ShellPath string
}

// 服务配置
type ServerModel struct {
	USBAutoMount string
}

type SystemConfig struct {
	ConfigPath string `json:"config_path"`
}
