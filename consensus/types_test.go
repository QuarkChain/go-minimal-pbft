package consensus

import (
	"bytes"
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
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

	b := &FullBlock{Block: types.NewBlock(
		&Header{
			ParentHash:     common.Hash{},
			Number:         big.NewInt(6),
			TimeMs:         34534,
			Coinbase:       common.Address{},
			LastCommitHash: common.Hash{},
			Difficulty:     big.NewInt(1),
			Extra:          []byte{},
			BaseFee:        big.NewInt(2), // TODO
			NextValidators: []common.Address{},
		},
		[]*types.Transaction{},
		[]*types.Header{},
		[]*types.Receipt{},
		trie.NewStackTrie(nil)),
		LastCommit: cm,
	}

	var buf bytes.Buffer
	err := b.EncodeRLP(&buf)
	assert.NoError(t, err)
	nb := FullBlock{}
	bs := buf.Bytes()
	err = nb.DecodeRLP(rlp.NewStream(bytes.NewReader(bs), 0))
	assert.NoError(t, err)
	assert.Equal(t, b.Hash(), nb.Hash())
	// assert.Equal(t, b.Body(), nb.Body())
	assert.Equal(t, b.LastCommit, nb.LastCommit)
}

func TestSerdeConsensusSyncRequest(t *testing.T) {
	req := &ConsensusSyncRequest{
		Height:           152,
		Round:            1356,
		HasProposal:      1,
		PrevotesBitmap:   []uint64{3},
		PrecommitsBitmap: []uint64{2},
	}

	bs, _ := rlp.EncodeToBytes(req)
	nreq := &ConsensusSyncRequest{}
	rlp.DecodeBytes(bs, nreq)
	assert.Equal(t, req, nreq)
}

func TestVoteSignBytes(t *testing.T) {
	v := Vote{
		Type:             PrecommitType,
		Height:           20,
		Round:            2,
		BlockID:          common.Hash{},
		TimestampMs:      352353,
		ValidatorAddress: common.Address{},
		ValidatorIndex:   3,
		Signature:        []byte{},
	}

	bs0 := v.VoteSignBytes("aaa")
	bs1 := v.VoteSignBytes("bbb")
	assert.NotEqual(t, bs0, bs1)

	v.Height = 21
	bs2 := v.VoteSignBytes("aaa")
	assert.NotEqual(t, bs0, bs2)
}

func TestSignVote(t *testing.T) {
	pv := GeneratePrivValidatorLocal()
	pubKey, err := pv.GetPubKey(context.Background())
	assert.NoError(t, err)

	v := Vote{
		Type:             PrecommitType,
		Height:           20,
		Round:            2,
		BlockID:          common.Hash{},
		TimestampMs:      352353,
		ValidatorAddress: pubKey.Address(),
		ValidatorIndex:   3,
		Signature:        []byte{},
	}

	pv.SignVote(context.Background(), "aaa", &v)
	assert.NoError(t, v.Verify("aaa", pubKey))
}

func TestSignProposal(t *testing.T) {
	pv := GeneratePrivValidatorLocal()
	pubKey, err := pv.GetPubKey(context.Background())
	assert.NoError(t, err)

	c := CommitSig{
		BlockIDFlag:      BlockIDFlagCommit,
		ValidatorAddress: common.Address{},
		TimestampMs:      1133423,
		Signature:        []byte{},
	}

	cm := NewCommit(5, 6, common.Hash{}, []CommitSig{c})

	proposal := Proposal{
		Height:      20,
		Round:       2,
		POLRound:    -1,
		TimestampMs: 352353,
		Signature:   []byte{},
		Block: &FullBlock{Block: types.NewBlock(
			&Header{
				ParentHash:     common.Hash{},
				Number:         big.NewInt(6),
				TimeMs:         34534,
				Coinbase:       common.Address{},
				LastCommitHash: common.Hash{},
				Difficulty:     big.NewInt(1),
				Extra:          []byte{},
				BaseFee:        big.NewInt(2), // TODO
				NextValidators: []common.Address{},
			},
			[]*types.Transaction{},
			[]*types.Header{},
			[]*types.Receipt{},
			trie.NewStackTrie(nil)),
			LastCommit: cm,
		},
	}

	pv.SignProposal(context.Background(), "aaa", &proposal)
	assert.True(t, pubKey.VerifySignature(proposal.ProposalSignBytes("aaa"), proposal.Signature))
}
