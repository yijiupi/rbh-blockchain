package main

import (
	"encoding/hex"
	"log"

	"github.com/boltdb/bolt"
)

const utxoBucket = "chainstate"

// 公共账本
type UTXOSet struct {
	BlockChain *BlockChain // 区块链
}

// 重建UTXO（保留未花费的输出）
func (u UTXOSet) Reindex() {
	db := u.BlockChain.db            // 获取数据库
	bucketName := []byte(utxoBucket) // 获取桶
	// 闭包内发现问题
	err := db.Update(func(tx *bolt.Tx) error { // 开始数据库事务
		err := tx.DeleteBucket(bucketName) // 删除桶
		if err != nil && err != bolt.ErrBucketNotFound {
			return err
		}
		_, err = tx.CreateBucket(bucketName) // 创建桶
		if err != nil {
			return err
		}
		return nil // 成功完成
	})
	// 闭包外处理问题
	if err != nil {
		log.Panic(err) // 回滚
	}

	UTXO := u.BlockChain.FindUTXO()
	// 闭包内发现问题
	err = db.Update(func(tx *bolt.Tx) error { // 开启事务二
		b := tx.Bucket(bucketName)     // 获取之前创建的桶
		for txID, outs := range UTXO { // 遍历utxo
			key, err := hex.DecodeString(txID) // 将交易ID从十六进制字符串解码为字节
			if err != nil {
				return err
			}

			err = b.Put(key, outs.Serialize()) // 将未花费输出序列化后存入桶
			if err != nil {
				return err
			}
		}
		return nil
	})
	// 闭包外处理问题
	if err != nil {
		log.Panic(err) // 回滚
	}
}
