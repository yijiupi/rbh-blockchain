package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/boltdb/bolt"
)

const utxoBucket = "chainstate"

// 公共账本
type UTXOSet struct {
	BlockChain *BlockChain // 区块链
}

// 重建UTXO（保留未花费的输出）
func (u UTXOSet) Reindex() error {
	db := u.BlockChain.db            // 获取数据库
	bucketName := []byte(utxoBucket) // 获取桶
	// 闭包内发现问题
	err := db.Update(func(tx *bolt.Tx) error { // 开始数据库事务
		err := tx.DeleteBucket(bucketName) // 删除桶
		if err := tx.DeleteBucket(bucketName); err != nil && err != bolt.ErrBucketNotFound {
			return err
		}
		_, err = tx.CreateBucket(bucketName) // 创建桶
		return err                           // 成功完成
	})
	// 闭包外处理问题
	if err != nil {
		return err
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
	return err
}

// 获取UTXO（公钥匹配得到UTXO，用于查询余额）
// GetUTXO 返回指定公钥哈希对应的所有 UTXO（未花费交易输出）。
// 参数 pubKeyHash 是公钥哈希（20 字节），用于匹配输出中的锁定脚本。
// 返回值：
//   - UTXOs 切片，包含所有匹配的 TxOutput。
//   - error 表示数据库操作过程中的错误（如 bucket 不存在或读取失败）。
func (u UTXOSet) GetUTXO(pubKeyHash []byte) ([]TxOutput, error) {
	var UTXOs []TxOutput  // 存储匹配的输出结果
	db := u.BlockChain.db // 获取区块链数据库实例

	// 使用只读事务查询数据库
	err := db.View(func(tx *bolt.Tx) error {
		// 获取 UTXO 桶（存储所有未花费输出）
		b := tx.Bucket([]byte(utxoBucket))
		if b == nil {
			// 桶不存在，返回明确的错误信息
			return fmt.Errorf("UTXO bucket not found")
		}

		// 创建游标遍历桶中所有键值对
		c := b.Cursor()
		// 遍历每一个交易输出集
		for k, v := c.First(); k != nil; k, v = c.Next() {
			// 反序列化输出集（该交易的所有输出）
			outs := DeserializeOutputs(v)

			// 遍历该交易中的每个输出
			for _, out := range outs.Outputs {
				// 检查当前输出是否被给定的公钥哈希锁定
				if out.IsLockedWithKey(pubKeyHash) {
					// 匹配成功，添加到结果集
					UTXOs = append(UTXOs, out)
				}
			}
		}

		return nil // 无错误，事务成功
	})

	// 返回结果和可能的错误（错误由调用方处理）
	return UTXOs, err
}

// 获取UTXO桶中所有的交易索引和未花费的金额（未花费的输出，用于转账）
func (u UTXOSet) GetUnSpendableOutputs(pubkeyHash []byte, amount uint64) (uint64, map[string][]int, error) {
	unspentOutputs := make(map[string][]int) // 未花费的输出
	var accumulated uint64                   // 未花费的金额（余额）
	db := u.BlockChain.db                    // 发送者的钱包数据库

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket)) // 获取UTXO桶
		if b == nil {
			return fmt.Errorf("UTXO bucket not found")
		}
		c := b.Cursor() // 遍历桶

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
	return accumulated, unspentOutputs, err
}

// 签名
func (bc *BlockChain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) error {
	prevTXs := make(map[string]Transaction) // 声明一个切片用于存储前面所有的交易的id
	// 遍历所有交易的输入，并按交易id查询交易，确保交易存在，并将之前所有交易id放入切片中
	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransaction(vin.Txid)
		if err != nil {
			return fmt.Errorf("find previous tx: %w", err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}
	return tx.Sign(privKey, prevTXs)
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
更新 UTXO 集合（基于新挖出的区块）
	步骤一当前交易的输入里面存在上一个交易的输出并且未花费的金额需要重新更新
	步骤二，只处理当前交易的输出
*/
// 返回值：error 表示更新过程中遇到的错误
func (u UTXOSet) Update(block *Block) error {
	db := u.BlockChain.db

	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		if b == nil {
			return fmt.Errorf("UTXO bucket not found")
		}

		// 步骤1：处理区块中所有交易的输入，删除/更新被花费的输出
		// 使用临时 map 记录每个交易 ID 的剩余输出（用于多个输入可能引用同一交易的情况）
		pendingUpdates := make(map[string][]TxOutput)

		for _, transaction := range block.Transactions {
			if transaction.IsCoinbase() {
				continue // Coinbase 交易没有输入，跳过
			}
			for _, vin := range transaction.Vin {
				txID := hex.EncodeToString(vin.Txid)

				// 从数据库获取该交易当前的 UTXO 集合
				outsBytes := b.Get(vin.Txid)
				if outsBytes == nil {
					return fmt.Errorf("UTXO not found for tx %s", txID)
				}
				outs := DeserializeOutputs(outsBytes)
				// 边界检查
				if int(vin.Vout) >= len(outs.Outputs) {
					return fmt.Errorf("Vout index %d out of range for tx %s", vin.Vout, txID)
				}
				// 构建该交易新的 UTXO 列表（移除被花费的输出）
				var remainingOuts []TxOutput
				for idx, out := range outs.Outputs {
					if uint64(idx) != vin.Vout {
						remainingOuts = append(remainingOuts, out)
					}
				}

				// 合并到 pendingUpdates 中
				if existing, ok := pendingUpdates[txID]; ok {
					// 将当前剩余输出追加到已有列表中
					pendingUpdates[txID] = append(existing, remainingOuts...)
				} else {
					pendingUpdates[txID] = remainingOuts
				}
			}
		}

		// 将 pendingUpdates 中的更新写回数据库
		for txIDHex, outputs := range pendingUpdates {
			txIDBytes, err := hex.DecodeString(txIDHex)
			if err != nil {
				return err
			}
			if len(outputs) == 0 {
				// 所有输出都被花费，删除该交易条目
				if err := b.Delete(txIDBytes); err != nil {
					return err
				}
			} else {
				// 更新该交易的 UTXO 条目
				updatedOuts := TxOutputs{Outputs: outputs}
				if err := b.Put(txIDBytes, updatedOuts.Serialize()); err != nil {
					return err
				}
			}
		}

		// 步骤2：处理区块中所有交易的输出，将新产生的 UTXO 添加到数据库
		for _, transaction := range block.Transactions {
			// 构建该交易的输出列表（切片）
			var newOutputs []TxOutput
			for _, out := range transaction.Vout {
				newOutputs = append(newOutputs, out)
			}
			// 包装为 TxOutputs 并写入数据库
			outsWrapper := TxOutputs{Outputs: newOutputs}
			if err := b.Put(transaction.ID, outsWrapper.Serialize()); err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

// 交易数量
func (u UTXOSet) CountTransactions() (int, error) {
	db := u.BlockChain.db
	counter := 0
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		if b == nil {
			return nil // 空桶也认为计数0
		}
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			counter++
		}
		return nil
	})
	return counter, err
}
