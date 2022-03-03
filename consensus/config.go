package consensus

import "time"

// ConsensusConfig defines the configuration for the Tendermint consensus service,
// including timeouts and details about the WAL and the block structure.
type ConsensusConfig struct {
	RootDir string `mapstructure:"home"`
	WalPath string `mapstructure:"wal-file"`
	WalFile string // overrides WalPath if set

	// TODO: remove timeout configs, these should be global not local
	// How long we wait for a proposal block before prevoting nil
	TimeoutPropose time.Duration `mapstructure:"timeout-propose"`
	// How much timeout-propose increases with each round
	TimeoutProposeDelta time.Duration `mapstructure:"timeout-propose-delta"`
	// How long we wait after receiving +2/3 prevotes for “anything” (ie. not a single block or nil)
	TimeoutPrevote time.Duration `mapstructure:"timeout-prevote"`
	// How much the timeout-prevote increases with each round
	TimeoutPrevoteDelta time.Duration `mapstructure:"timeout-prevote-delta"`
	// How long we wait after receiving +2/3 precommits for “anything” (ie. not a single block or nil)
	TimeoutPrecommit time.Duration `mapstructure:"timeout-precommit"`
	// How much the timeout-precommit increases with each round
	TimeoutPrecommitDelta time.Duration `mapstructure:"timeout-precommit-delta"`
	// How long we wait after committing a block, before starting on the new
	// height (this gives us a chance to receive some more precommits, even
	// though we already have +2/3).
	TimeoutCommit time.Duration `mapstructure:"timeout-commit"`

	// Make progress as soon as we have all the precommits (as if TimeoutCommit = 0)
	SkipTimeoutCommit bool `mapstructure:"skip-timeout-commit"`

	// Reactor sleep duration parameters
	PeerGossipSleepDuration     time.Duration `mapstructure:"peer-gossip-sleep-duration"`
	PeerQueryMaj23SleepDuration time.Duration `mapstructure:"peer-query-maj23-sleep-duration"`

	ConsensusSyncRequestDuration time.Duration

	DoubleSignCheckHeight uint64 `mapstructure:"double-sign-check-height"`
}

func NewDefaultConsesusConfig() *ConsensusConfig {
	// Config from tendermint except TimeoutCommit is 5s (original 1s)
	return &ConsensusConfig{
		// WalPath:                     filepath.Join(defaultDataDir, "cs.wal", "wal"),
		TimeoutPropose:               3000 * time.Millisecond,
		TimeoutProposeDelta:          500 * time.Millisecond,
		TimeoutPrevote:               1000 * time.Millisecond,
		TimeoutPrevoteDelta:          500 * time.Millisecond,
		TimeoutPrecommit:             1000 * time.Millisecond,
		TimeoutPrecommitDelta:        500 * time.Millisecond,
		TimeoutCommit:                5000 * time.Millisecond,
		SkipTimeoutCommit:            false,
		PeerGossipSleepDuration:      100 * time.Millisecond,
		PeerQueryMaj23SleepDuration:  2000 * time.Millisecond,
		DoubleSignCheckHeight:        uint64(0),
		ConsensusSyncRequestDuration: 500 * time.Millisecond,
	}
}

// Propose returns the amount of time to wait for a proposal
func (cfg *ConsensusConfig) Propose(round int32) time.Duration {
	return time.Duration(
		cfg.TimeoutPropose.Nanoseconds()+cfg.TimeoutProposeDelta.Nanoseconds()*int64(round),
	) * time.Nanosecond
}

// Prevote returns the amount of time to wait for straggler votes after receiving any +2/3 prevotes
func (cfg *ConsensusConfig) Prevote(round int32) time.Duration {
	return time.Duration(
		cfg.TimeoutPrevote.Nanoseconds()+cfg.TimeoutPrevoteDelta.Nanoseconds()*int64(round),
	) * time.Nanosecond
}

// Precommit returns the amount of time to wait for straggler votes after receiving any +2/3 precommits
func (cfg *ConsensusConfig) Precommit(round int32) time.Duration {
	return time.Duration(
		cfg.TimeoutPrecommit.Nanoseconds()+cfg.TimeoutPrecommitDelta.Nanoseconds()*int64(round),
	) * time.Nanosecond
}

// Commit returns the amount of time to wait for straggler votes after receiving +2/3 precommits
// for a single block (ie. a commit).
func (cfg *ConsensusConfig) Commit(t time.Time) time.Time {
	return t.Add(cfg.TimeoutCommit)
}
