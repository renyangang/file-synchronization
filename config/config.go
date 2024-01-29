package config

import (
	"path/filepath"

	"stacktrace.top/filesync/logger"

	"github.com/spf13/viper"
)

type Config struct {
	Sync SyncConfig
}

type SyncConfig struct {
	Srcpath      string
	Dstpath      string
	Cachefile    string
	Dstcachefile string
	Excludefrom  string
}

var ServerConfig Config

func init() {
	v := viper.New()
	v.SetConfigName("config")
	v.AddConfigPath(".")
	v.SetConfigType("toml")
	if err := v.ReadInConfig(); err != nil {
		logger.Error("read config failed.err:%s", err)
	}
	if err := v.Unmarshal(&ServerConfig); err != nil {
		logger.Error("make config obj failed.err:%s", err)
	}
	ServerConfig.Sync.Srcpath = filepath.Clean(ServerConfig.Sync.Srcpath)
	logger.Info("config read:%v", ServerConfig)
}
