//go:build !windows
// +build !windows

package config

func GetDefaultConfigDir() string {
	return "/etc/harbor"
}

func GetDefaultPluginDirectory() string {
	return "/etc/harbor/plugins"
}
