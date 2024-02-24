package sync

import (
	"os"
	"path/filepath"
	"sync"

	"stacktrace.top/filesync/config"
	"stacktrace.top/filesync/logger"
)

type SyncOper interface {
	// 对比目录
	CompareDiffFiles() (map[string]*SyncFileInfo, error)
	// 同步文件
	SyncFile(srcFilePath string, dstFilePath string, fileInfo *SyncFileInfo) error

	SyncFiles(diffFiles map[string]*SyncFileInfo)
}

type OsSyncOper struct {
}

func (o *OsSyncOper) CompareDiffFiles() (map[string]*SyncFileInfo, error) {
	LoadSrcCache()
	loadDstCache()
	return Compare(), nil
}

func (o *OsSyncOper) SyncFiles(diffFiles map[string]*SyncFileInfo) {
	var wg sync.WaitGroup
	for fp, fi := range diffFiles {
		wg.Add(1)
		go func(filePath string, fileInfo *SyncFileInfo) {
			defer wg.Done()
			logger.Info("sync file: %v", filePath)
			srcFilePath := filepath.Join(config.InstanceConfig.Sync.Srcpath, filePath)
			dstFilePath := filepath.Join(config.InstanceConfig.Sync.Dstpath, filePath)
			err := o.SyncFile(srcFilePath, dstFilePath, fileInfo)
			if err == nil {
				DstSyncFileMap[filePath] = fileInfo
			}
		}(fp, fi)
	}
	wg.Wait()
}

func (o *OsSyncOper) SyncFile(srcFilePath string, dstFilePath string, fileInfo *SyncFileInfo) error {
	if fileInfo.IsDir {
		err := os.MkdirAll(dstFilePath, fileInfo.Mode)
		if err != nil {
			logger.Error("create dir: %v failed.err: %v", dstFilePath, err)
			return err
		}
	} else {
		data, err := os.ReadFile(srcFilePath)
		if err != nil {
			logger.Error("read file: %v failed.err: %v", srcFilePath, err)
			return err
		}
		path := filepath.Dir(dstFilePath)
		os.MkdirAll(path, os.ModePerm)
		err = os.WriteFile(dstFilePath, data, fileInfo.Mode)
		if err != nil {
			logger.Error("write file: %v failed.err: %v", dstFilePath, err)
			return err
		}
		err = os.Chtimes(dstFilePath, fileInfo.ModTime, fileInfo.ModTime)
		if err != nil {
			logger.Error("change file: %v time failed.err: %v", dstFilePath, err)
			return err
		}
	}
	return nil
}
