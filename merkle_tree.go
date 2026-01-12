package main

// 默克尔树
type MerkleTree struct {
	RootNode *MerkleNode // 根节点
}
// 默克尔节点
type MerkleNode struct {
	Left  *MerkleTree // 左叶子
	Rigth *MerkleTree // 右叶子
	Data  []byte      // 交易数据
}
