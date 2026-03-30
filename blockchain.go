package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/boltdb/bolt"
)

const dbFile = "blockchain_%s.db"
const blocksBucket = "blocks"
const genesisCoinbaseData = ""

var (
	ErrBlockchainNotExist = fmt.Errorf("blockchain database not found")
	ErrBucketNotFound     = fmt.Errorf("blocks bucket not found")
	ErrTipNotFound        = fmt.Errorf("tip not found")
)

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
		return nil, ErrBlockchainNotExist
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
	var lastHeight uint64 = 0
	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket)) // 从当前事务中获取名为 'blocks' 的 bucket
		if b == nil {
			return ErrBucketNotFound
		}
		lastHash := b.Get([]byte("l")) // 从 bucket 中获取键为 'l' 的值，最后一个区块的hash
		if lastHash == nil {
			// 没有区块，高度保持为 0
			return nil
		}
		blockData := b.Get(lastHash) // 通过hash获取到区块的数据的二进制
		if blockData == nil {
			return fmt.Errorf("block data missing")
		}
		block := DeserializeBlock(blockData) // 反序列化得到区块数据
		lastHeight = block.Height
		return nil
	})
	if err != nil {
		return 0, err
	}
	return lastHeight, nil
}

// 创建一个新区块链
func CreateBlockchain(address, nodeID string) (*BlockChain, error) {
	dbFile := fmt.Sprintf(dbFile, nodeID) // 数据库文件名
	if dbExists(dbFile) {                 // 是否存在，存在退出
		return nil, fmt.Errorf("blockchain already exists")
	}

	cbtx, err := NewCoinbaseTX(address, genesisCoinbaseData) // 新建创世交易
	if err != nil {
		return nil, fmt.Errorf("create coinbase tx: %w", err)
	}
	genesis := NewGenesisBlock(cbtx) // 新建创世区块

	db, err := bolt.Open(dbFile, 0600, nil) // 打开数据库读写权限
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	err = db.Update(func(tx *bolt.Tx) error { // 开启事务
		b, err := tx.CreateBucket([]byte(blocksBucket)) // 创建桶
		if err != nil {
			return err
		}
		if err := b.Put(genesis.Hash, genesis.Serialize()); err != nil { // 区块入桶
			return err
		}
		if err := b.Put([]byte("l"), genesis.Hash); err != nil { // 区块hash入桶key为l
			return err
		}
		return nil //结束事务
	})
	if err != nil {
		db.Close() // 关闭数据库
		return nil, fmt.Errorf("update db: %w", err)
	}

	bc := BlockChain{genesis.Hash, db} // 创建区块链，传入区块hash和数据库指针
	return &bc, nil
}

// CreateEmptyBlockchain 创建空的区块链数据库（无区块）
func CreateEmptyBlockchain(nodeID string) (*BlockChain, error) {
	dbFile := fmt.Sprintf(dbFile, nodeID)
	if dbExists(dbFile) {
		return nil, fmt.Errorf("database already exists")
	}
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		// 创建 blocks 桶
		if _, err := tx.CreateBucketIfNotExists([]byte(blocksBucket)); err != nil {
			return err
		}
		// 创建 chainstate 桶（UTXO）
		if _, err := tx.CreateBucketIfNotExists([]byte(utxoBucket)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create buckets: %w", err)
	}
	return &BlockChain{newBlockHash: nil, db: db}, nil
}

// initBlockchain 初始化区块链（若不存在则创建空链）
func initBlockchain(nodeID string) (*BlockChain, error) {
	bc, err := GetBlockchain(nodeID)
	if err != nil {
		if errors.Is(err, ErrBlockchainNotExist) {
			return CreateEmptyBlockchain(nodeID)
		}
		return nil, err
	}
	return bc, nil
}

// 创世区块交易
func NewCoinbaseTX(to, data string) (*Transaction, error) {
	if data == "" {
		randData := make([]byte, 20)  // 创建一个20字节的字节切片
		_, err := rand.Read(randData) // 随机数写入20个字节
		if err != nil {
			return nil, fmt.Errorf("failed to generate random data for coinbase: %w", err)
		}
		data = fmt.Sprintf("%x", randData) // 十六进制字符串，公钥
	}
	txin := TxInput{[]byte{}, 0, nil, []byte(data)}             // 交易输入初始化，带公钥
	txout := NewTxOutput(10, to)                                // 奖励接收者，10个币
	tx := Transaction{nil, []TxInput{txin}, []TxOutput{*txout}} // 输入输出组成交易
	tx.ID = tx.Hash()
	return &tx, nil
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
			return nil, fmt.Errorf("verify transaction: %w", err)
		} else if !ok {
			return nil, fmt.Errorf("invalid transaction")
		}
	}
	// 数据库找到区块，并得到区块的hash
	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte("l"))
		if lastHash == nil {
			return fmt.Errorf("no blocks in chain")
		}
		blockData := b.Get(lastHash)
		if blockData == nil {
			return fmt.Errorf("block data missing")
		}
		block := DeserializeBlock(blockData)
		lastHeight = block.Height
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

	// 清理内存池和广播记录（一次性完成）
	// 注意：锁顺序必须统一（先 broadcastedTxsMutex，后 memPoolMutex），避免死锁
	broadcastedTxsMutex.Lock()
	memPoolMutex.Lock()
	for _, tx := range newBlock.Transactions {
		txID := hex.EncodeToString(tx.ID)
		delete(memPool, txID)
		delete(broadcastedTxs, txID)
	}
	memPoolMutex.Unlock()
	broadcastedTxsMutex.Unlock()

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
func NewBlockchain(nodeID string) (*BlockChain, error) {
	dbFile := fmt.Sprintf(dbFile, nodeID) // blockchain_modeIDxxxx.db组装字符串
	if !dbExists(dbFile) {
		return nil, fmt.Errorf("no existing blockchain found. create one first")
	}

	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			return fmt.Errorf("blocks bucket not found")
		}
		tip = b.Get([]byte("l"))
		if tip == nil {
			return fmt.Errorf("no tip found")
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("read tip: %w", err)
	}

	bc := BlockChain{tip, db}
	return &bc, nil
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
