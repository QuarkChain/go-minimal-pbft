package main

import (
	"github.com/ethereum/go-ethereum/common"
)

type Node struct {
	inChan  chan interface{}
	network *Network

	currentBlock *Block

	validators []common.Address

	self common.Address
}

type Network struct {
	nodes []*Node
}

type Block struct {
	height   int
	prevHash [32]byte
}

type Vote struct {
	block Block
	round int
	sign  []byte
}

func NewNode(self common.Address, validators []common.Address) *Node {
	return &Node{
		self:       self,
		validators: validators,
	}
}

func (n *Node) Run() {
	for {
		select {
		case msg := <-n.inChan:
			switch msg.(type) {

			}
		}
	}
}

func (n *Node) IsProposer(height int) bool {
	for i, vaddr := range n.validators {
		if vaddr == n.self {
			return (i+height)%len(n.validators) == 0
		}
	}
	return false
}

func (n *Node) Broadcast(msg interface{}) {
	for _, node := range n.network.nodes {
		if node == n {
			continue
		}

		go func(node *Node) {
			// TODO: make a copy of msg?
			node.inChan <- msg
		}(node)
	}
}
