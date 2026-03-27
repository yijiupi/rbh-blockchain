package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/boltdb/bolt"
)

const dbFile = "blockchain_%s.db"
const blocksBucket = "blocks"
const genesisCoinbaseData = ""

// 区块链
type BlockChain struct {
	newBlockHash []byte   // 最新区块的hash
	db           *bolt.DB // 数据库
}
type BlockchainIterator struct {
	currentHash []byte
	db          *bolt.DB
}

func (i *BlockchainIterator) Next() *Block {
	var block *Block
	// 开始一个数据库只读事务（View），用于从数据库中读取数据
	err := i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))   //获取存储区块的桶（bucket）
		encodedBlock := b.Get(i.currentHash)   //使用当前哈希作为键，从数据库中获取序列化的区块数据
		block = DeserializeBlock(encodedBlock) //调用 DeserializeBlock 函数，将序列化的字节数据反序列化为 Block 结构体

		return nil //事务成功完成
	})

	if err != nil {
		log.Panic(err) //错误回滚
	}
	// 向前遍历，每次获取一个区块后，将 currentHash 设置为该区块的前驱区块哈希
	i.currentHash = block.PrevBlockHash

	return block
}

// 获取当前节点的区块链（区块的hash和数据库）
func GetBlockchain(nodeID string) (*BlockChain, error) {
	dbFile := fmt.Sprintf(dbFile, nodeID) // blockchain_modeIDxxxx.db组装字符串
	if !dbExists(dbFile) {
		return nil, fmt.Errorf("区块链节点文件不存在，可以尝试创建一个")
	}
	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil) // 打开数据库文件blockchain_3000.db得到数据库db
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error { // 启动读写事务
		b := tx.Bucket([]byte(blocksBucket)) // 从当前事务中获取名为 'blocks' 的 bucket
		tip = b.Get([]byte("l"))             // 从 bucket 中获取键为 'l' 的值，最后一个区块的hash
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	bc := BlockChain{tip, db}
	return &bc, nil
}

// 判定blockchain_3000.db文件是否存在，若不存在返回一个错误err；判断错误err是否是文件不存在
func dbExists(dbFile string) bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

// 获取区块高度
func (bc *BlockChain) GetBestHeight() (uint64, error) {
	var lastBlock Block

	err := bc.db.View(func(tx *bolt.Tx) error { // 数据库中查看
		b := tx.Bucket([]byte(blocksBucket)) // 从当前事务中获取名为 'blocks' 的 bucket
		lastHash := b.Get([]byte("l"))       // 从 bucket 中获取键为 'l' 的值，最后一个区块的hash
		if lastHash == nil {
			return fmt.Errorf("no block found")
		}
		blockData := b.Get(lastHash) // 通过hash获取到区块的数据的二进制
		if blockData == nil {
			return fmt.Errorf("block data missing")
		}
		lastBlock = *DeserializeBlock(blockData) // 反序列化得到区块数据
		return nil
	})
	if err != nil {
		return 0, err
	}
	return lastBlock.Height, nil
}

// 创建一个新区块链
func CreateBlockchain(address, nodeID string) *BlockChain {
	dbFile := fmt.Sprintf(dbFile, nodeID) // 数据库文件名
	if dbExists(dbFile) {                 // 文件是否存在
		fmt.Println("Blockchain already exists.") // 文件已经存在退出
		os.Exit(1)
	}
	var newBlockHash []byte                             // 声明新区块的hash
	cbtx := NewCoinbaseTX(address, genesisCoinbaseData) // 新建创世区块交易
	genesis := NewGenesisBlock(cbtx)                    // 新建创世区块

	db, err := bolt.Open(dbFile, 0600, nil) // 打开数据库文件读写权限
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error { // 开启数据库事务
		b, err := tx.CreateBucket([]byte(blocksBucket)) // 创建bucket
		if err != nil {
			log.Panic(err)
		}

		err = b.Put(genesis.Hash, genesis.Serialize()) // 区块入桶key为区块hash值为区块
		if err != nil {
			log.Panic(err)
		}

		err = b.Put([]byte("l"), genesis.Hash) // 区块hash入桶，key为l，值为区块hash
		if err != nil {
			log.Panic(err)
		}
		newBlockHash = genesis.Hash // 将创世区块的哈希设置为当前最新区块哈希

		return nil // 结束事务
	})
	if err != nil { // 检查事务执行是否出错
		log.Panic(err)
	}

	bc := BlockChain{newBlockHash, db} // 最新区块哈希和数据库连接

	return &bc
}

// 创世区块交易
func NewCoinbaseTX(to, data string) *Transaction {
	if data == "" {
		randData := make([]byte, 20)  // 创建一个20字节的字节切片
		_, err := rand.Read(randData) // 随机数写入20个字节
		if err != nil {
			log.Panic(err) // 如果生成随机数失败，终止程序
		}

		data = fmt.Sprintf("%x", randData) // 十六进制字符串，公钥
	}

	txin := TxInput{[]byte{}, 0, nil, []byte(data)}             // 交易输入初始化，带公钥
	txout := NewTxOutput(10, to)                                // 奖励接收者，10个币
	tx := Transaction{nil, []TxInput{txin}, []TxOutput{*txout}} // 输入输出组成交易
	tx.ID = tx.Hash()

	return &tx
}

// UTXO查找
func (bc *BlockChain) FindUTXO() map[string]TxOutputs {
	UTXO := make(map[string]TxOutputs)     // 创建空的 UTXO 映射，用于存储未花费的输出
	spentTXOs := make(map[string][]uint64) // 创建已花费输出映射。键是交易ID，值是已花费输出的索引列表（Vout索引）
	bci := bc.Iterator()                   // 获取区块链迭代器

	for {
		block := bci.Next() // 内部（向前遍历，将迭代器的当前哈希更新为刚获取的区块的前一个区块哈希，直至创世区块）
		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)
		Outputs:
			for outIdx, out := range tx.Vout {
				// txID交易下，已花费输出索引，交易输出的索引，相等说明已花费，不相等代表未花费，当前输出加入UTXO
				if spentTXOs[txID] != nil {
					for _, spentOutIdx := range spentTXOs[txID] {
						if spentOutIdx == uint64(outIdx) {
							continue Outputs
						}
					}
				}

				outs := UTXO[txID]                       // 初始化输出变量
				outs.Outputs = append(outs.Outputs, out) // 变量内部参数，追加
				UTXO[txID] = outs                        // 把输出变量赋值
			}

			if tx.IsCoinbase() == false { // 不是创世交易，需要获取到上一个交易的input输入
				for _, in := range tx.Vin {
					inTxID := hex.EncodeToString(in.Txid)                  // 上一个交易的id
					spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout) // 将输出加入到已花费列表
				}
			}
		}
		// 到达创世区块结束
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return UTXO
}

// 初始化区块链迭代器对象（这里只是初始化区块链迭代器对象，其实与区块链对象无差别，只是为了隔离）
func (bc *BlockChain) Iterator() *BlockchainIterator {
	bci := &BlockchainIterator{bc.newBlockHash, bc.db}
	return bci
}

// 挖出区块交易和转账交易数组，放入一个新区块
func (bc *BlockChain) MineBlock(transactions []*Transaction) (*Block, error) {
	var lastHash []byte
	var lastHeight uint64
	// 挖矿和转账交易的验证
	// TODO: ignore transaction if it's not valid
	// 这里面验证的是，是否为上一个区块的交易，显然不是的，这里是挖到的新区块和新转账交易，其实不用验证这里是为了保险
	for _, tx := range transactions {
		if ok, err := bc.VerifyTransaction(tx); err != nil {
			return nil, fmt.Errorf("verify transaction error: %w", err)
		} else if !ok {
			return nil, fmt.Errorf("invalid transaction")
		}
	}
	// 数据库找到区块，并得到区块的hash
	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte("l")) // 最后的区块hash
		if lastHash == nil {
			return fmt.Errorf("no blocks in chain")
		}
		blockData := b.Get(lastHash) // 最后的区块数据
		if blockData == nil {
			return fmt.Errorf("block data missing")
		}
		block := DeserializeBlock(blockData) // 反序列化区块数据得到区块
		lastHeight = block.Height            // 得到最后区块的高度

		return nil
	})
	if err != nil {
		return nil, err
	}
	// 挖到的新区块高度加一，加入交易列表，放在新区块上
	newBlock := NewBlock(transactions, lastHash, lastHeight+1)
	// 新区块存储数据库
	err = bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if err := b.Put(newBlock.Hash, newBlock.Serialize()); err != nil {
			return err
		}
		if err := b.Put([]byte("l"), newBlock.Hash); err != nil {
			return err
		}
		bc.newBlockHash = newBlock.Hash
		return nil
	})
	if err != nil {
		return nil, err
	}
	return newBlock, nil
}

// 验证交易（验证上一个区块的交易）
func (bc *BlockChain) VerifyTransaction(tx *Transaction) (bool, error) {
	if tx.IsCoinbase() {
		return true, nil
	}
	prevTXs := make(map[string]Transaction)
	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransaction(vin.Txid)
		if err != nil {
			return false, fmt.Errorf("find previous tx: %w", err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}
	return tx.Verify(prevTXs), nil
}

// 通过节点获取当前区块链
func NewBlockchain(nodeID string) *BlockChain {
	dbFile := fmt.Sprintf(dbFile, nodeID) // blockchain_modeIDxxxx.db组装字符串
	if dbExists(dbFile) == false {        // 这个文件是否存在？
		fmt.Println("No existing blockchain found. Create one first.")
		os.Exit(1)
	}

	var newBlockHash []byte
	db, err := bolt.Open(dbFile, 0600, nil) // 开启数据库
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error { // 启动读写事务
		b := tx.Bucket([]byte(blocksBucket)) // 设置bucket的值为blocks,bucket是kv集合数据库
		newBlockHash = b.Get([]byte("l"))    // 获取bucket中的值，新区块的hash

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	bc := BlockChain{newBlockHash, db}

	return &bc
}

// AddBlock 将已验证的区块添加到区块链（包括数据库存储）
func (bc *BlockChain) AddBlock(block *Block) error {
	err := bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			return fmt.Errorf("blocks bucket not found")
		}

		// 检查区块是否已存在（可选，避免重复写入）
		existing := b.Get(block.Hash)
		if existing != nil {
			return nil // 区块已存在，跳过
		}

		// 序列化并存储区块
		err := b.Put(block.Hash, block.Serialize())
		if err != nil {
			return err
		}

		// 更新最新区块哈希（'l' 键）
		err = b.Put([]byte("l"), block.Hash)
		if err != nil {
			return err
		}

		bc.newBlockHash = block.Hash
		return nil
	})
	return err
}

// FindBlock 根据区块哈希查找区块，返回区块和错误
func (bc *BlockChain) FindBlock(hash []byte) (*Block, error) {
	var block *Block
	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			return fmt.Errorf("blocks bucket not found")
		}
		data := b.Get(hash)
		if data == nil {
			return fmt.Errorf("block %x not found", hash)
		}
		block = DeserializeBlock(data)
		return nil
	})
	return block, err
}
