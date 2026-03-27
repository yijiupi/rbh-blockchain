package main

import (
	"bytes"
	"fmt"
	"math/big"
)

var b58Alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

// Base58Decode 解码 Base58 字符串，返回字节数组和错误
func Base58Decode(input []byte) ([]byte, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("empty input")
	}
	result := big.NewInt(0)
	for _, b := range input {
		charIndex := bytes.IndexByte(b58Alphabet, b)
		if charIndex < 0 {
			return nil, fmt.Errorf("invalid base58 character: %c", b)
		}
		result.Mul(result, big.NewInt(58))
		result.Add(result, big.NewInt(int64(charIndex)))
	}
	decoded := result.Bytes()
	// 处理前导零（统计开头的 '1' 数量）
	zeroCount := 0
	for zeroCount < len(input) && input[zeroCount] == b58Alphabet[0] {
		zeroCount++
	}
	// 在解码结果前添加对应数量的 0x00
	decoded = append(bytes.Repeat([]byte{0x00}, zeroCount), decoded...)
	return decoded, nil
}

// Base58Encode 将字节数组编码为 Base58 字符串
func Base58Encode(input []byte) []byte {
	var result []byte
	x := big.NewInt(0).SetBytes(input)
	base := big.NewInt(int64(len(b58Alphabet)))
	zero := big.NewInt(0)
	mod := &big.Int{}
	for x.Cmp(zero) != 0 {
		x.DivMod(x, base, mod)
		result = append(result, b58Alphabet[mod.Int64()])
	}
	// 处理前导零
	for _, b := range input {
		if b != 0x00 {
			break
		}
		result = append(result, b58Alphabet[0])
	}
	ReverseBytes(result)
	return result
}

func ReverseBytes(data []byte) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}
