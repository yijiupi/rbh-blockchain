# 区块链节点启动与同步完整流程

## 环境准备
- 项目已编译：
  - Windows：`go build -o rbh-blockchain.exe`
  - Linux/macOS：`go build -o rbh-blockchain`
- 准备三个终端窗口，分别对应节点：
  - 终端1：种子节点（端口 3000）
  - 终端2：普通节点（端口 3001）
  - 终端3：普通节点（端口 3002）

---

## 完整操作命令（按顺序执行）

以下所有命令按终端和步骤排列，请在对应的终端窗口中依次执行。


# ========================= 终端1：种子节点（3000） =========================

# 设置节点ID（根据操作系统选择）
# Windows PowerShell
    $env:NODE_ID="3000"
# Linux/macOS
# export NODE_ID=3000
# Windows CMD
# set NODE_ID=3000

# 1. 创建钱包（得到地址A，请记录）
./rbh-blockchain createwallet

# 2. 创建创世区块链，奖励给地址A
./rbh-blockchain createblockchain -address <地址A>

# 3. 启动种子节点（监听端口，等待其他节点连接）
./rbh-blockchain startnode

# ========================= 终端2：普通节点（3001） =========================

# 设置节点ID
    $env:NODE_ID="3001"   # Windows PowerShell
# export NODE_ID=3001 # Linux/macOS

# 可选：创建自己的钱包（地址B，用于接收转账）
./rbh-blockchain createwallet

# 注意：不要执行 createblockchain（否则会创建独立的创世区块）
# 直接启动节点
./rbh-blockchain startnode

# ========================= 终端3：普通节点（3002） =========================

# 设置节点ID
    $env:NODE_ID="3002"   # Windows PowerShell
# export NODE_ID=3002 # Linux/macOS

# 创建钱包（地址C）
./rbh-blockchain createwallet

# 注意：不要执行 createblockchain（否则会创建独立的创世区块）
# 直接启动节点
./rbh-blockchain startnode

# ========================= 验证同步状态（在任意终端执行） =========================

# 打印整个区块链信息
./rbh-blockchain printchain
# 查询节点下的所有钱包地址
.\rbh-blockchain.exe listaddresses

# 查看各地址余额
./rbh-blockchain getbalance -address <地址A>   # 应显示 10（创世奖励）
./rbh-blockchain getbalance -address <地址B>   # 应显示 0（尚未收到转账）

# ========================= 转账测试（在种子节点 3000 的终端执行） =========================

./rbh-blockchain send -from <地址A> -to <地址B> -amount 2 -mine

# ========================= 重置所有数据（彻底重新开始） =========================

# 关闭所有节点：在每个终端按 Ctrl + C
go clean -cache   # 彻底清理缓存
go build -o rbh-blockchain.exe
# 删除所有区块链数据库和钱包文件
# Windows PowerShell
Remove-Item *.db, *.db.lock, wallet_*.dat
# Linux/macOS
# rm -f *.db wallet_*.dat