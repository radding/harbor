package config

import "os"

func GetDefaultConfigDir() string {
	return os.ExpandEnv("${APPDATA}\\harbor")
}

func GetDefaultPluginDirectory() string {
	return os.ExpandEnv(("${ProgramFiles}\\harbor\\plugins"))
}
