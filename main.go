package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"time"

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
	logger.Info("need sync files: %v", len(diffFiles))
	return diffFiles
}

func syncFiles(diffFiles map[string]*sync.SyncFileInfo) {
	syncOper := makeSyncOper()
	syncOper.SyncFiles(diffFiles)
}

func DoSync() {
	diffFiles := CompareDiffFiles()
	// mySyncFiles := make(map[string]*sync.SyncFileInfo)
	// for k, v := range diffFiles {
	// 	mySyncFiles[k] = v
	// 	break
	// }
	syncFiles(diffFiles)
}

// 全局变量，用于存储 recover() 返回的值
var panicValue interface{}

// 全局变量，用于存储 panic 发生时的堆栈信息
var panicStack []byte

// 全局函数，用于恢复 panic 并记录相关信息
func globalRecover() {
	if r := recover(); r != nil {
		// 保存 panic 的值和堆栈信息
		panicValue = r
		panicStack = make([]byte, 1024*1024) // 分配足够的空间来保存堆栈信息
		n := runtime.Stack(panicStack, false)
		panicStack = panicStack[:n]
		fmt.Println("Recovered from panic:", r, panicStack)
	}
}

func procSignal() {
	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	c := make(chan os.Signal, 1)

	// Passing no signals to Notify means that
	// all signals will be sent to the channel.
	signal.Notify(c)

	// Block until any signal is received.
	s := <-c
	fmt.Println("Got signal:", s)
}

func main() {
	defer globalRecover()
	go procSignal()
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
		case "daemon":
			net.StartServer()
		default:
			fmt.Println("usage: filesync makecache | compare | sync")
		}
	} else {
		fmt.Println("usage: filesync makecache | compare | sync")
	}
	// 程序正常退出
	time.Sleep(time.Second * 100)
	logger.Info("filesync exited")
}
