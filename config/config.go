package config

import (
	"path/filepath"

	"stacktrace.top/filesync/logger"

	"github.com/spf13/viper"
)

type Config struct {
	Sync   SyncConfig
	Server ServerConfig
	Client ClientConfig
}

type SyncConfig struct {
	Srcpath      string
	Dstpath      string
	Cachefile    string
	Dstcachefile string
	Excludefrom  string
}

type ServerConfig struct {
	Port        int
	Token       string
	Compression bool
	Workers     int
	Blocksize   int
}

type ClientConfig struct {
	Serverip   string
	Serverport int
	Token      string
	Threads    int
}

var InstanceConfig Config

func init() {
	v := viper.New()
	v.SetConfigName("config")
	v.AddConfigPath(".")
	v.SetConfigType("toml")
	if err := v.ReadInConfig(); err != nil {
		logger.Error("read config failed.err:%s", err)
	}
	if err := v.Unmarshal(&InstanceConfig); err != nil {
		logger.Error("make config obj failed.err:%s", err)
	}
	InstanceConfig.Sync.Srcpath = filepath.Clean(InstanceConfig.Sync.Srcpath)
	InstanceConfig.Server.Blocksize *= (1024 * 1024)
	logger.Info("config read:%v", InstanceConfig)
}
