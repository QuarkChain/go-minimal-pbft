package consensus

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
)

func TestSerdeCommitSig(t *testing.T) {
	c := CommitSig{
		BlockIDFlag:      BlockIDFlagCommit,
		ValidatorAddress: common.Address{},
		TimestampMs:      1133423,
		Signature:        []byte{},
	}

	data, err := rlp.EncodeToBytes(c)
	assert.NoError(t, err)
	nc := CommitSig{}
	err = rlp.DecodeBytes(data, &nc)
	assert.NoError(t, err)
	assert.Equal(t, c, nc)
}

func TestSerdeCommit(t *testing.T) {
	c := CommitSig{
		BlockIDFlag:      BlockIDFlagCommit,
		ValidatorAddress: common.Address{},
		TimestampMs:      1133423,
		Signature:        []byte{},
	}
	cm := NewCommit(5, 6, common.Hash{}, []CommitSig{c})

	data, err := rlp.EncodeToBytes(cm)
	assert.NoError(t, err)
	ncm := Commit{}
	err = rlp.DecodeBytes(data, &ncm)
	assert.NoError(t, err)
	assert.Equal(t, *cm, ncm)
}

func TestSerdeBlock(t *testing.T) {
	c := CommitSig{
		BlockIDFlag:      BlockIDFlagCommit,
		ValidatorAddress: common.Address{},
		TimestampMs:      1133423,
		Signature:        []byte{},
	}
	cm := NewCommit(5, 6, common.Hash{}, []CommitSig{c})
	b := Block{
		LastBlockID:     common.Hash{},
		Height:          6,
		TimeMs:          34534,
		ProposerAddress: common.Address{},
		LastCommit:      cm,
	}

	data, err := rlp.EncodeToBytes(b)
	assert.NoError(t, err)
	nb := Block{}
	err = rlp.DecodeBytes(data, &nb)
	assert.NoError(t, err)
	assert.Equal(t, b, nb)
}
