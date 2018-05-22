package main

import (
	"time"
	"fmt"
	"0522/PoSdemo/Block"
	"0522/PoSdemo/BlockChain"
	"math/rand"
)

//全节点
type Node struct {

	tokens int

	adds string
}

//存放全节点
var n [2]Node

//
var addr[3000]string


func main()  {

	//存放两个炒币者
	n[0] = Node{1000,"abc123"}

	n[1] = Node{2000,"bcd234"}


	//以下为PoS算法
	var cnt = 0
	for i := 0; i < 2; i++ {
		for j := 0; j < n[i].tokens; j++ {
			addr[cnt] = n[i].adds
			cnt++
		}
	}


	rand.Seed(time.Now().Unix())
	var rd = rand.Intn(3000)
	var adds = addr[rd]

	var firstBlcok Block.Block
	firstBlcok.Index = 1
	firstBlcok.BMP = 100
	firstBlcok.PrefHash = "0"
	firstBlcok.TimeStamp = time.Now().String()
	firstBlcok.Validator = "abc123"
	firstBlcok.HashCode = BlockChain.GenerateHashValue(firstBlcok)


	//将创世区块添加到区块链
	BlockChain.BlockChain = append(BlockChain.BlockChain, firstBlcok)
	//挖矿成功
	var seoundBlock = BlockChain.GenerateNextBlock(firstBlcok,200,adds)

	BlockChain.BlockChain = append(BlockChain.BlockChain,seoundBlock)

	fmt.Println(BlockChain.BlockChain)

}
