package main

import (
	"fmt"
	"os"

	"stacktrace.top/filesync/sync"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "makecache":
			// 执行makecache操作
			sync.MakeSrcInfo()
		case "compare":
			// 执行compare操作
			sync.CompareDiffFiles()
		case "sync":
			// 执行sync操作
			sync.DoSync()
		default:
			fmt.Println("usage: filesync makecache | compare | sync")
		}
	} else {
		fmt.Println("usage: filesync makecache | compare | sync")
	}
}
