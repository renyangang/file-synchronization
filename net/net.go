package net

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"stacktrace.top/filesync/config"
	"stacktrace.top/filesync/logger"
	"stacktrace.top/filesync/sync"
)

type SyncInfo struct {
	FilePath string
	FileInfo *sync.SyncFileInfo
}

type SyncServer struct {
	conn    *net.TCPConn
	running bool
}

type SyncClient struct {
	conn           *net.TCPConn
	dstFileInfoMap map[string]*sync.SyncFileInfo
	infoChan       chan *SyncInfo
}

func (syncServer *SyncServer) Stop() {
	syncServer.running = false
	syncServer.conn.Close()
}

func (syncServer *SyncServer) Loop() {
	defer syncServer.Stop()
	syncServer.running = true
	syncServer.conn.SetReadDeadline(time.Now().Add(time.Second * 10))
	msg, err := ReadForSyncMsg(syncServer.conn)
	if err != nil {
		logger.Error("read token msg error: %v", err)
		return
	}
	if msg.MsgType != MSG_TOKEN {
		logger.Error("first msg is not token")
		return
	}
	if msg.Token != config.InstanceConfig.Server.Token {
		logger.Error("token is invalid")
		return
	}
	resMsg := &SyncRespMsg{
		MsgType:   msg.MsgType,
		ResCode:   RES_SUCCESS,
		Err:       nil,
		FileInfos: nil,
	}
	syncServer.response(resMsg)
	syncServer.conn.SetReadDeadline(time.Now().AddDate(10, 0, 0))
	for syncServer.running {
		msg, err = ReadForSyncMsg(syncServer.conn)
		if netErr, ok := err.(net.Error); ok {
			if !netErr.Timeout() {
				syncServer.Stop()
				break
			}
		}
		if err != nil {
			logger.Error("read msg error: %v", err)
			continue
		}
		syncServer.ProcMsg(msg)
	}
}

func (syncServer *SyncServer) ProcMsg(msg *SyncCmdMsg) {
	switch msg.MsgType {
	case MSG_MAKECACHE:
		syncServer.makeCache(msg)
	case MSG_SYNC:
		syncServer.sync(msg)
	default:
		logger.Error("unknown msg type: %d", msg.MsgType)
	}
}

func (syncServer *SyncServer) response(resMsg *SyncRespMsg) {
	err := WriteForSyncRespMsg(syncServer.conn, resMsg)
	if netErr, ok := err.(net.Error); ok {
		if !netErr.Timeout() {
			syncServer.Stop()
		}
	}
}

func (syncServer *SyncServer) makeCache(msg *SyncCmdMsg) {
	resMsg := &SyncRespMsg{
		MsgType:   msg.MsgType,
		ResCode:   RES_SUCCESS,
		Err:       nil,
		FileInfos: nil,
	}
	if msg.DstDir == "" {
		resMsg.ResCode = 1
		resMsg.Err = errors.New("no dst dir provide")
	} else {
		cacheMap := sync.MakeDirInfo(msg.DstDir)
		resMsg.FileInfos = cacheMap
	}
	syncServer.response(resMsg)
}

func (syncServer *SyncServer) sync(msg *SyncCmdMsg) {
	resMsg := &SyncRespMsg{
		MsgType:   msg.MsgType,
		ResCode:   RES_SUCCESS,
		Err:       nil,
		FileInfos: nil,
	}
	if msg.DstDir == "" || msg.SyncInfo == nil {
		resMsg.ResCode = 1
		resMsg.Err = errors.New("no dst dir or syncFileInfo provide")
	} else {
		if msg.SyncInfo.IsDir {
			err := os.MkdirAll(msg.DstDir, msg.SyncInfo.Mode)
			if err != nil {
				logger.Error("create dir: %v failed.err: %v", msg.DstDir, err)
			}
		} else {
			totalSize := msg.SyncInfo.Size
			bufSize := config.InstanceConfig.Server.Blocksize
			if totalSize < int64(config.InstanceConfig.Server.Blocksize) {
				bufSize = int(totalSize)
			}
			revBuffer := make([]byte, bufSize)
			path := filepath.Dir(msg.DstDir)
			os.MkdirAll(path, os.ModePerm)
			file, err := os.OpenFile(msg.DstDir, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, msg.SyncInfo.Mode)
			if err != nil {
				logger.Error("open file failed. file: %v, err: %v", msg.DstDir, err)
				resMsg.ResCode = 1
				resMsg.Err = errors.New("open file failed")
				syncServer.response(resMsg)
				return
			}
			defer file.Close()
			offset := int64(0)
			for offset < totalSize {
				fileResMsg := &SyncRespMsg{
					MsgType:  MSG_FILEPART,
					OffSet:   offset,
					PartSize: int64(config.InstanceConfig.Server.Blocksize),
				}
				syncServer.response(fileResMsg)
				n, err := syncServer.conn.Read(revBuffer)
				if err != nil {
					logger.Error("read file content failed. err: %v", err)
					syncServer.Stop()
					return
				}
				file.Write(revBuffer[:n])
				offset += int64(n)
			}
		}
		err := os.Chtimes(msg.DstDir, msg.SyncInfo.ModTime, msg.SyncInfo.ModTime)
		if err != nil {
			logger.Error("change file: %v time failed.err: %v", msg.DstDir, err)
		}
	}
	syncServer.response(resMsg)
}

func StartServer() error {
	addr := &net.TCPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: config.InstanceConfig.Server.Port,
	}
	server, err := net.ListenTCP("tcp", addr)
	if err != nil {
		logger.Error("start server failed. port: %v, err: %v", config.InstanceConfig.Server.Port, err)
		return err
	}
	for {
		conn, err := server.AcceptTCP()
		if err != nil {
			logger.Error("accept conn from server failed, err: %v", config.InstanceConfig.Server.Port, err)
			return err
		}
		syncServer := &SyncServer{
			conn: conn,
		}
		go syncServer.Loop()
	}
}

func StartClient() (*SyncClient, error) {
	addr := &net.TCPAddr{
		IP:   net.ParseIP(config.InstanceConfig.Client.Serverip),
		Port: config.InstanceConfig.Client.Serverport,
	}
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		logger.Error("connect server failed. err: %v", err)
		return nil, err
	}
	syncClient := &SyncClient{
		conn: conn,
	}
	err = syncClient.SendToken()
	if err != nil {
		return nil, err
	}
	return syncClient, nil
}

func (sc *SyncClient) SendToken() error {
	return SendToken(sc.conn)
}

func SendToken(conn *net.TCPConn) error {
	msg := &SyncCmdMsg{
		MsgType: MSG_TOKEN,
		Token:   config.InstanceConfig.Client.Token,
	}
	err := WriteForSyncMsg(conn, msg)
	if err != nil {
		logger.Error("send token failed. err: %v", err)
		return err
	}
	resMsg, err := ReadForSyncRespMsg(conn)
	if err != nil {
		logger.Error("read token response failed. err: %v", err)
		return err
	}
	if resMsg.MsgType != MSG_TOKEN || resMsg.ResCode != RES_SUCCESS {
		logger.Error("Token msg error: %v", resMsg)
		return errors.New("token msg error")
	}
	return nil
}

func (sc *SyncClient) CompareDiffFiles() (map[string]*sync.SyncFileInfo, error) {
	msg := &SyncCmdMsg{
		MsgType: MSG_MAKECACHE,
		DstDir:  config.InstanceConfig.Sync.Dstpath,
	}
	err := WriteForSyncMsg(sc.conn, msg)
	if err != nil {
		logger.Error("send compare info failed. err: %v", err)
		return nil, err
	}
	resMsg, err := ReadForSyncRespMsg(sc.conn)
	if err != nil {
		logger.Error("read compare response failed. err: %v", err)
		return nil, err
	}
	sync.DstSyncFileMap = resMsg.FileInfos
	sync.LoadSrcCache()
	return sync.Compare(), nil
}

func (sc *SyncClient) SyncFiles(diffFiles map[string]*sync.SyncFileInfo) {
	sc.infoChan = make(chan *SyncInfo, config.InstanceConfig.Client.Threads)
	resChan := make(chan int, len(diffFiles))
	go func() {
		for i := 0; i < config.InstanceConfig.Client.Threads; i++ {
			go func() {
				scFile, err := StartClient()
				if err != nil {
					logger.Error("start client failed. err: %v", err)
					return
				}
				for sInfo := range sc.infoChan {
					srcFilePath := filepath.Join(config.InstanceConfig.Sync.Srcpath, sInfo.FilePath)
					dstFilePath := filepath.Join(config.InstanceConfig.Sync.Dstpath, sInfo.FilePath)
					err := scFile.SyncFile(srcFilePath, dstFilePath, sInfo.FileInfo)
					if err != nil {
						logger.Error("sync file failed. err: %v", err)
					}
					resChan <- 1
				}
			}()
		}
	}()
	for fp, fi := range diffFiles {
		// 同步文件
		sInfo := &SyncInfo{
			FilePath: fp,
			FileInfo: fi,
		}
		sc.infoChan <- sInfo
	}
	sum := 0
	for n := range resChan {
		sum += n
		if sum >= len(diffFiles) {
			break
		}
	}
}

func (sc *SyncClient) SyncFile(srcFilePath string, dstFilePath string, fileInfo *sync.SyncFileInfo) error {
	msg := &SyncCmdMsg{
		MsgType:  MSG_SYNC,
		DstDir:   dstFilePath,
		SyncInfo: fileInfo,
	}
	err := WriteForSyncMsg(sc.conn, msg)
	if err != nil {
		return err
	}
	for {
		// 等待分片请求
		resMsg, err := ReadForSyncRespMsg(sc.conn)
		if err != nil {
			return err
		}
		if resMsg.MsgType == MSG_FILEPART {
			// 发送分片
			file, err := os.Open(srcFilePath)
			if err != nil {
				logger.Error("open file failed. file: %v, err: %v", srcFilePath, err)
				return err
			}
			buf := make([]byte, resMsg.PartSize)
			size, err := file.ReadAt(buf, resMsg.OffSet)
			if err != nil {
				logger.Error("read file failed. file: %v, err: %v", srcFilePath, err)
				return err
			}
			sc.conn.Write(buf[:size])
		} else if resMsg.MsgType == MSG_SYNC && resMsg.ResCode != RES_SUCCESS {
			return fmt.Errorf("sync file failed. file: %v, res: %v", srcFilePath, resMsg.ResCode)
		} else {
			break
		}
	}
	return nil
}
