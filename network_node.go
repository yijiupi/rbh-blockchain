package main

import (
	"log"
	"math/rand"
	"time"
)

// 下载管理器实现
func downloadManager(bc *BlockChain) {
	ticker := time.NewTicker(10 * time.Second) // 定期检查
	for range ticker.C {
		downloadMutex.Lock()
		// 1. 检查超时：遍历 downloadingBlocks，超时则重试
		now := time.Now()
		for hash, startTime := range downloadingBlocks {
			if now.Sub(startTime) > 30*time.Second {
				// 超时，重试
				retries := downloadRetries[hash] + 1
				if retries >= 3 {
					// 放弃该区块，从待下载中删除
					delete(blocksToDownload, hash)
					delete(downloadingBlocks, hash)
					delete(downloadRetries, hash)
					log.Printf("Failed to download block %s after 3 retries", hash)
				} else {
					// 重新请求
					downloadRetries[hash] = retries
					// 选择节点
					node := selectNodeForBlock()
					sendGetData(node, "block", []byte(hash))
					downloadingBlocks[hash] = now // 更新时间
				}
			}
		}

		// 2. 从待下载队列中取区块开始下载
		for hash := range blocksToDownload {
			if _, downloading := downloadingBlocks[hash]; !downloading {
				// 尚未下载，选择节点发送请求
				node := selectNodeForBlock()
				sendGetData(node, "block", []byte(hash))
				downloadingBlocks[hash] = time.Now()
				delete(blocksToDownload, hash)
				// 同时将该哈希移出待下载队列，避免重复
			}
		}
		downloadMutex.Unlock()
	}
}

// 选择节点函数（简单随机或轮询）
func selectNodeForBlock() string {
	knownNodesMutex.RLock()
	defer knownNodesMutex.RUnlock()
	if len(knownNodes) == 0 {
		return ""
	}
	// 简单随机选择
	idx := rand.Intn(len(knownNodes))
	return knownNodes[idx]
}
