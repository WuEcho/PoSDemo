package BlockChain

import (
	"PoSdemo/Block"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

//创建区块链
var BlockChain []Block.Block

//计算区块的哈希
func GenerateHashValue(block Block.Block) string {

	var hashCode = block.PrefHash + block.TimeStamp + block.Validator + strconv.Itoa(block.BMP) +
		strconv.Itoa(block.Index)

	var shar = sha256.New()
	shar.Write([]byte(hashCode))

	return hex.EncodeToString(shar.Sum(nil))
}

//生成新区快的函数   adds - 地址 (矿工的地址)
func GenerateNextBlock(oldBlock Block.Block, BMP int, adds string) Block.Block {

	var newBlcok Block.Block
	newBlcok.Index = oldBlock.Index + 1
	newBlcok.TimeStamp = time.Now().String()
	newBlcok.BMP = BMP
	newBlcok.Validator = adds
	newBlcok.PrefHash = oldBlock.HashCode
	newBlcok.HashCode = GenerateHashValue(newBlcok)
	return newBlcok
}
