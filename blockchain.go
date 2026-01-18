package main

import (
	"fmt"
	"log"
	"os"

	"github.com/boltdb/bolt"
)

const dbFile = "blockchain_%s.db"
const blocksBucket = "blocks"

// 区块链
type BlockChain struct {
	newBlockHash []byte   // 最新区块的hash
	db           *bolt.DB // 数据库
}

// 获取当前节点的区块链（区块的hash和数据库）
func GetBlockchain(nodeID string) *BlockChain {
	dbFile := fmt.Sprintf(dbFile, nodeID) // blockchain_modeIDxxxx.db组装字符串
	if dbExists(dbFile) == false {        // 判定blockchain_3000.db文件是否存在？
		fmt.Println("区块链节点文件不存在，可以尝试创建一个")
		os.Exit(1)
	}
	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil) // 打开数据库文件blockchain_3000.db得到数据库db
	if err != nil {
		log.Panic(err)
	}
	err = db.Update(func(tx *bolt.Tx) error { // 启动读写事务
		b := tx.Bucket([]byte(blocksBucket)) // 从当前事务中获取名为 'blocks' 的 bucket
		tip = b.Get([]byte("l"))             // 从 bucket 中获取键为 'l' 的值，最后一个区块的hash
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	bc := BlockChain{tip, db}
	return &bc
}

// 判定blockchain_3000.db文件是否存在，若不存在返回一个错误err；判断错误err是否是文件不存在
func dbExists(dbFile string) bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

// 获取区块高度
func (bc *BlockChain) GetBestHeight() uint64 {
	var lastBlock Block

	err := bc.db.View(func(tx *bolt.Tx) error { // 数据库中查看
		b := tx.Bucket([]byte(blocksBucket))     // 从当前事务中获取名为 'blocks' 的 bucket
		lastHash := b.Get([]byte("l"))           // 从 bucket 中获取键为 'l' 的值，最后一个区块的hash
		blockData := b.Get(lastHash)             // 通过hash获取到区块的数据的二进制
		lastBlock = *DeserializeBlock(blockData) // 反序列化得到区块数据

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return lastBlock.Height
}
