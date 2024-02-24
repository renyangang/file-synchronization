package main

import (
	"fmt"
	"os"

	"stacktrace.top/filesync/config"
	"stacktrace.top/filesync/logger"
	"stacktrace.top/filesync/net"
	"stacktrace.top/filesync/sync"
)

func makeSyncOper() sync.SyncOper {
	switch config.InstanceConfig.Sync.Syncmode {
	case config.LOCAL_MODE:
		return &sync.OsSyncOper{}
	case config.NET_MODE:
		sc, err := net.StartClient()
		if err != nil {
			logger.Error("StartClient failed. Error: %v", err)
			os.Exit(1)
		}
		return sc
	default:
		return &sync.OsSyncOper{}
	}
}

func CompareDiffFiles() map[string]*sync.SyncFileInfo {
	syncOper := makeSyncOper()
	diffFiles, err := syncOper.CompareDiffFiles()
	if err != nil {
		logger.Error("CompareDiffFiles failed. Error: %v", err)
		return nil
	}
	return diffFiles
}

func syncFiles(diffFiles map[string]*sync.SyncFileInfo) {
	syncOper := makeSyncOper()
	syncOper.SyncFiles(diffFiles)
}

func DoSync() {
	diffFiles := CompareDiffFiles()
	syncFiles(diffFiles)
}

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "makecache":
			// 执行makecache操作
			sync.MakeSrcInfo()
		case "compare":
			// 执行compare操作
			CompareDiffFiles()
		case "sync":
			// 执行sync操作
			DoSync()
		default:
			fmt.Println("usage: filesync makecache | compare | sync")
		}
	} else {
		fmt.Println("usage: filesync makecache | compare | sync")
	}
}
