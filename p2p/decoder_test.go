package p2p

import (
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-minimal-pbft/consensus"
	"github.com/stretchr/testify/assert"
)

func TestSerdeProposal(t *testing.T) {
	p := &consensus.Proposal{
		Height:    4,
		Round:     3,
		POLRound:  -1,
		Timestamp: time.Now().UnixMilli(),
		BlockID:   common.Hash{},
		Signature: []byte{},
	}

	data, err := encodeProposal(p)
	fmt.Println(data)
	assert.NoError(t, err)
	np, err := decodeProposal(data)
	assert.NoError(t, err)
	assert.Equal(t, p, np)
}
