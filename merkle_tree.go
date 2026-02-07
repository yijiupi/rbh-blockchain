package main

import "crypto/sha256"

// 默克尔树
type MerkleTree struct {
	RootNode *MerkleNode // 根节点
}

// 默克尔节点
type MerkleNode struct {
	Left  *MerkleNode // 左叶子
	Right *MerkleNode // 右叶子
	Data  []byte      // 交易数据
}

// 新建一个默克尔树（当前交易数据加入默克尔树，最终返回根节点）
func NewMerkleTree(data [][]byte) *MerkleTree {
	var nodes []MerkleNode
	// 奇数检查，例如交易数据集合为: [A, B, C] (3个，奇数)复制后: [A, B, C, C] (4个，偶数)
	if len(data)%2 != 0 {
		data = append(data, data[len(data)-1])
	}

	for _, datum := range data {
		node := NewMerkleNode(nil, nil, datum) // 新建节点（交易数据加密，得到hash放入节点的data）
		nodes = append(nodes, *node)           // 默克尔节点的指针合集[HashA, HashB, HashC, HashC]
	}
	// 开始构建树的循环。循环次数为 len(data)/2，因为每轮会将节点数量减半。
	for i := 0; i < len(data)/2; i++ {
		var newLevel []MerkleNode // 声明一个新的切片，用于存储当前层构建出的父节点
		// 循环遍历当前层的节点，每次取2个，左右节点[AB, CC]
		for j := 0; j < len(nodes); j += 2 {
			node := NewMerkleNode(&nodes[j], &nodes[j+1], nil) // 新建节点（左右节点交易数据拼接后加密，得到hash）
			newLevel = append(newLevel, *node)                 // 左右节点的指针作为父节点[HashAB, HashCC]
		}
		// [HashA, HashB, HashC, HashC]替换为[HashAB, HashCC]在替换为[HashABCC]作为根节点
		nodes = newLevel
	}

	mTree := MerkleTree{&nodes[0]}

	return &mTree
}

// 新建默克尔树节点（左右节点合并成新节点作为父节点返回）
func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	mNode := MerkleNode{} // 左右节点合并成新节点作为父节点
	// 检查左右子节点是否都为空。如果都为空，那么这个节点就是一个叶子节点；如果不是叶子节点，非叶子节点都有两个子节点
	if left == nil && right == nil {
		hash := sha256.Sum256(data) // 交易数据加密，得到hash
		mNode.Data = hash[:]
	} else {
		prevHashes := append(left.Data, right.Data...) // 左右节点的数据拼接
		hash := sha256.Sum256(prevHashes)              // 左右节点交易数据拼接后加密，得到hash
		mNode.Data = hash[:]
	}

	mNode.Left = left
	mNode.Right = right

	return &mNode
}
