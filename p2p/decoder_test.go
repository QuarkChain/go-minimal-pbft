package p2p

import (
	"math/big"
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
		Signature: []byte{'1'},
	}

	data, err := encodeProposal(p)
	assert.NoError(t, err)
	np, err := decodeProposal(data)
	assert.NoError(t, err)
	assert.Equal(t, p, np)
}

func TestSerdeVote(t *testing.T) {
	v := &consensus.Vote{
		Type:             consensus.PrevoteType,
		Height:           4,
		Round:            3,
		Timestamp:        time.Now().UnixMilli(),
		BlockID:          common.Hash{},
		ValidatorAddress: common.BigToAddress(big.NewInt(12345)),
		ValidatorIndex:   5,
		Signature:        []byte{'2'},
	}

	data, err := encodeVote(v)
	assert.NoError(t, err)
	nv, err := decodeVote(data)
	assert.NoError(t, err)
	assert.Equal(t, v, nv)
}
