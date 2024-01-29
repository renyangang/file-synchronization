package main

import (
	"os"

	"stacktrace.top/filesync/logger"
)

type SyncOper interface {
	// 同步文件
	SyncFile(srcFilePath string, dstFilePath string, fileInfo *SyncFileInfo) error
}

type OsSyncOper struct {
}

func (o *OsSyncOper) SyncFile(srcFilePath string, dstFilePath string, fileInfo *SyncFileInfo) error {
	if fileInfo.IsDir {
		err := os.MkdirAll(dstFilePath, fileInfo.Mode)
		if err != nil {
			logger.Info("create dir: %v failed.err: %v", dstFilePath, err)
			return err
		}
	} else {
		data, err := os.ReadFile(srcFilePath)
		if err != nil {
			logger.Info("read file: %v failed.err: %v", srcFilePath, err)
			return err
		}
		err = os.WriteFile(dstFilePath, data, fileInfo.Mode)
		if err != nil {
			logger.Info("write file: %v failed.err: %v", dstFilePath, err)
			return err
		}
		err = os.Chtimes(dstFilePath, fileInfo.ModTime, fileInfo.ModTime)
		if err != nil {
			logger.Info("change file: %v time failed.err: %v", dstFilePath, err)
			return err
		}
	}
	return nil
}
