package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
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

// 获取UTXO（公钥匹配得到UTXO，用于查询余额）
func (u UTXOSet) GetUTXO(pubKeyHash []byte) []TxOutput {
	var UTXOs []TxOutput
	db := u.BlockChain.db
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor() // 创建游标，遍历桶中数据
		// k为交易ID从十六进制字符串解码为字节   v为TxOutputs.Serialize
		for k, v := c.First(); k != nil; k, v = c.Next() {
			outs := DeserializeOutputs(v)

			for _, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) { // 公钥匹配得到UTXO
					UTXOs = append(UTXOs, out)
				}
			}
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return UTXOs
}

// 获取UTXO桶中所有的交易索引和未花费的金额（未花费的输出，用于转账）
func (u UTXOSet) GetUnSpendableOutputs(pubkeyHash []byte, amount uint64) (uint64, map[string][]int) {
	unspentOutputs := make(map[string][]int) // 未花费的输出
	var accumulated uint64                   // 未花费的金额（余额）
	db := u.BlockChain.db                    // 发送者的钱包数据库

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket)) // 获取UTXO桶
		c := b.Cursor()                    // 遍历桶

		for k, v := c.First(); k != nil; k, v = c.Next() {
			txID := hex.EncodeToString(k) // 解密交易id
			outs := DeserializeOutputs(v) // 反序列化交易输出集合的结构体

			for outIdx, out := range outs.Outputs { // 获取每一笔交易的所有输出
				// 单个UTXO可能面值不足，当 accumulated >= amount 时，说明已经收集够了，可以停止搜索
				if out.IsLockedWithKey(pubkeyHash) && accumulated < amount {
					accumulated += out.Value
					/*
						unspentOutputs = {
							"交易ID1": [输出索引1, 输出索引2, ...],
							"交易ID2": [输出索引1, 输出索引3, ...],
							"交易ID3": [输出索引0, 输出索引2, 输出索引4, ...],
							...
						}
					*/
					unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)
				}
			}
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return accumulated, unspentOutputs
}

// 签名
func (bc *BlockChain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction) // 声明上一个交易

	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransaction(vin.Txid) // 查询上一个交易
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	tx.Sign(privKey, prevTXs)
}

// 根据交易id查询交易
func (bc *BlockChain) FindTransaction(ID []byte) (Transaction, error) {
	bci := bc.Iterator()

	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			if bytes.Equal(tx.ID, ID) {
				return *tx, nil // 在区块中找到了上一次交易
			}
		}
		// 未找上一次交易的hash
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("Transaction is not found")
}

/*
更新UTXO
（分两部分，第一部分当前交易的输入里面存在上一个交易的输出并且未花费的金额需要重新更新，
第二部分，只处理当前交易的输出）
*/
func (u UTXOSet) Update(block *Block) {
	db := u.BlockChain.db

	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		// 交易列表（新区块和转账交易）内部处理输入部分
		for _, tx := range block.Transactions {
			if tx.IsCoinbase() == false { // 如果不是Coinbase交易（Coinbase交易没有输入）
				for _, vin := range tx.Vin { // 遍历当前交易的每个输入
					updatedOuts := TxOutputs{}            // 创建一个空的输出集合，用于处理上一个交易的输出，未花费的输出
					outsBytes := b.Get(vin.Txid)          // 根据输入交易id找到，上一个交易的输出
					outs := DeserializeOutputs(outsBytes) // 反序列化输出

					for outIdx, out := range outs.Outputs { // 遍历输出
						// 上一个交易的输出和当前交易输入里的输出一致才会呗消费，否则未被消费放入未来的交易余额里
						if uint64(outIdx) != vin.Vout {
							updatedOuts.Outputs = append(updatedOuts.Outputs, out) // 未花费的交易要协会数据库
						}
					}
					// 判断更新后的输出列表是否为空
					if len(updatedOuts.Outputs) == 0 {
						// 如果所有输出都花费了，删除整个交易条目
						err := b.Delete(vin.Txid)
						if err != nil {
							log.Panic(err)
						}
					} else {
						// 还有未花费的输出，更新数据库
						err := b.Put(vin.Txid, updatedOuts.Serialize())
						if err != nil {
							log.Panic(err)
						}
					}

				}
			}
			// 外部值处理输出部分，一旦为创世交易不需要输入所以这里只处理输出，输入有上面处理
			newOutputs := TxOutputs{} // 新的交易输出声明
			for _, out := range tx.Vout {
				newOutputs.Outputs = append(newOutputs.Outputs, out)
			}
			// 将交易的输出存入数据库，以便下一次交易查询到
			err := b.Put(tx.ID, newOutputs.Serialize())
			if err != nil {
				log.Panic(err)
			}
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}

// 交易数量
func (u UTXOSet) CountTransactions() int {
	db := u.BlockChain.db
	counter := 0

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			counter++
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return counter
}
