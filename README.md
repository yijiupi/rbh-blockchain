# RBH Blockchain

## 项目概述
本项目是一个基于Go语言实现的完整区块链系统，具备以下核心功能：
- 区块与链的创建、持久化存储
- 工作量证明（PoW）共识机制
- 基于UTXO的交易模型及签名验证
- 钱包管理（地址生成、私钥存储）
- P2P网络节点发现与区块链同步
- 命令行交互界面

项目共包含24个源代码文件，采用模块化设计，各模块职责清晰，适合学习区块链底层原理。

---

## 核心模块解析

### 1. 区块与区块链（block.go, blockchain.go）
- **Block结构**：包含高度、Nonce、交易列表、前后哈希、时间戳。
- **创世区块**：通过`NewGenesisBlock`生成，内置一笔Coinbase交易。
- **区块链存储**：使用BoltDB（嵌入式键值数据库）持久化区块，键为区块哈希，值为序列化后的区块数据。
- **迭代器**：`BlockchainIterator`支持从最新区块向创世区块遍历。
- **主要方法**：`NewBlock`、`AddBlock`、`FindBlock`、`GetBestHeight`等。

### 2. 工作量证明（pow.go）
- **难度设置**：`targetBits = 16`，目标哈希值前导零的个数较低（便于演示）。
- **挖矿过程**：`ProofOfWork.Run()`循环递增nonce，计算区块头+默克尔根+时间戳等组合的SHA256，直到哈希值小于目标值。
- **验证机制**：`Validate()`检查区块是否满足难度要求。

### 3. 交易与UTXO（transaction.go, tx_input.go, tx_output.go, utxo_set.go）
- **UTXO模型**：`UTXOSet`管理未花费输出，独立存储在`chainstate`桶中，避免每次查询遍历整个区块链。
- **交易结构**：
  - `TxInput`：引用前序交易输出，包含签名和公钥。
  - `TxOutput`：包含金额和公钥哈希（锁定脚本）。
- **交易签名**：使用ECDSA（椭圆曲线），签名时复制“修剪版”交易（清空签名和公钥）进行哈希。
- **Coinbase交易**：挖矿奖励交易，无输入，输出给矿工地址。
- **UTXO更新**：`UTXOSet.Update()`在新块加入时，删除被花费的输出，添加新产生的输出。

### 4. 钱包与地址（wallet.go, wallets.go, base58.go）
- **钱包结构**：存储ECDSA私钥和公钥。
- **地址生成**：`version(0x00) + RIPEMD160(SHA256(公钥)) + 前4字节校验和`，经Base58编码。
- **地址验证**：通过解码和校验和比对。
- **钱包集合**：以文件形式存储（`wallet_<nodeID>.dat`），支持多钱包管理。

### 5. P2P网络（server.go 及各类handler）
- **协议**：TCP，消息格式为`[12字节命令][序列化数据]`。
- **消息类型**：
  - `version`：握手，交换区块高度。
  - `getblocks` / `inv`：请求/通告区块哈希列表。
  - `getdata` / `block` / `tx`：请求/传输具体区块或交易。
  - `addr`：节点地址广播。
- **节点角色**：种子节点（localhost:3000）作为引导，其他节点加入后同步区块链。
- **同步逻辑**：通过`version`比较高度，决定主动请求或被动发送区块；使用`blocksInTransit`队列顺序下载区块。
- **内存池**：`memPool`存储未确认交易，广播到网络。

### 6. 命令行接口（cli.go, main.go）
- **支持命令**：
  - `createwallet`：生成新钱包地址。
  - `getbalance -address <addr>`：查询余额。
  - `listaddresses`：列出所有地址。
  - `createblockchain -address <addr>`：初始化创世区块并奖励给该地址。
  - `printchain`：打印所有区块。
  - `reindexutxo`：重建UTXO集。
  - `send -from <addr> -to <addr> -amount <n> -mine`：转账，可立即挖矿或广播。
  - `startnode -miner <addr>`：启动节点，若指定矿工地址则开启挖矿。
- **节点标识**：通过环境变量`NODE_ID`区分不同节点（如3000、3001等），确保文件不冲突。

### 7. 默克尔树（merkle.go）
- 实现`MerkleTree`和`MerkleNode`，用于将交易列表压缩为一个根哈希，简化区块头验证。
- 当交易数为奇数时，复制最后一个交易以保证左右子节点配对。

---

## 设计亮点
1. **模块化清晰**：每个核心功能独立成文件，便于维护和扩展。
2. **持久化可靠**：BoltDB提供事务性操作，保证数据一致性。
3. **UTXO高效存储**：独立UTXO集避免全链扫描，提升查询性能。
4. **P2P消息处理**：统一在`handleConnection`中根据命令分发，易于添加新消息类型。
5. **命令行友好**：使用flag包解析参数，提供直观的操作命令。

---

## 总结
该项目是一个完整的、教学级别的区块链实现，涵盖了比特币的核心技术：PoW、UTXO、交易签名、P2P网络、钱包地址等。代码结构清晰，适合区块链入门学习。若用于生产环境，需在并发控制、网络稳定性、错误处理等方面加强优化。
