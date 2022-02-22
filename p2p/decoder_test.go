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
	c := consensus.CommitSig{
		BlockIDFlag:      consensus.BlockIDFlagCommit,
		ValidatorAddress: common.Address{},
		TimestampMs:      1133423,
		Signature:        []byte{},
	}
	cm := consensus.NewCommit(5, 6, common.Hash{}, []consensus.CommitSig{c})

	p := &consensus.Proposal{
		Height:      4,
		Round:       3,
		POLRound:    -1,
		TimestampMs: time.Now().UnixMilli(),
		BlockID:     common.Hash{},
		Signature:   []byte{'1'},
		Block: &consensus.Block{
			Header: consensus.Header{
				ParentHash:     common.Hash{},
				Number:         6,
				TimeMs:         34534,
				Coinbase:       common.Address{},
				LastCommitHash: common.Hash{},
			},
			Data:       []byte{},
			LastCommit: cm,
		},
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
		TimestampMs:      uint64(time.Now().UnixMilli()),
		BlockID:          common.BytesToHash([]byte{1, 2}),
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
