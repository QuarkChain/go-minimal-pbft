package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-minimal-pbft/consensus"
	"github.com/go-minimal-pbft/p2p"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"

	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

var (
	p2pNetworkID *string
	p2pPort      *uint
	p2pBootstrap *string
	nodeKeyPath  *string
	valKeyPath   *string
	nodeName     *string
	verbosity    *int
	datadir      *string
	validatorSet *[]string
)

var NodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Run the bridge server",
	Run:   runNode,
}

func init() {
	p2pNetworkID = NodeCmd.Flags().String("network", "/mpbft/dev", "P2P network identifier")
	p2pPort = NodeCmd.Flags().Uint("port", 8999, "P2P UDP listener port")
	p2pBootstrap = NodeCmd.Flags().String("bootstrap", "", "P2P bootstrap peers (comma-separated)")

	nodeName = NodeCmd.Flags().String("nodeName", "", "Node name to announce in gossip heartbeats")
	nodeKeyPath = NodeCmd.Flags().String("nodeKey", "", "Path to node key (will be generated if it doesn't exist)")

	verbosity = NodeCmd.Flags().Int("verbosity", 3, "Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail")

	valKeyPath = NodeCmd.Flags().String("valKey", "", "Path to validator key (empty if not a validator)")

	datadir = NodeCmd.Flags().String("datadir", "./datadir", "Path to database")

	validatorSet = NodeCmd.Flags().StringArray("validatorSet", []string{}, "List of validators")

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	// setup logger
	var ostream log.Handler
	output := io.Writer(os.Stderr)

	usecolor := (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"
	if usecolor {
		output = colorable.NewColorableStderr()
	}
	ostream = log.StreamHandler(output, log.TerminalFormat(usecolor))

	glogger.SetHandler(ostream)

	// logging
	glogger.Verbosity(log.Lvl(*verbosity))
}

func runNode(cmd *cobra.Command, args []string) {
	// Node's main lifecycle context.
	rootCtx, rootCtxCancel := context.WithCancel(context.Background())
	defer rootCtxCancel()

	// Outbound gossip message queue
	sendC := make(chan *consensus.Message)

	// Inbound observations
	obsvC := make(chan *consensus.MsgInfo, 50)

	// Load p2p private key
	var priv p2pcrypto.PrivKey
	priv, err := getOrCreateNodeKey(*nodeKeyPath)
	if err != nil {
		log.Error("Failed to load node key", "err", err)
		return
	}

	if *nodeKeyPath == "" {
		log.Error("Please specify --nodeKey")
	}

	var privVal consensus.PrivValidator
	var pubVal consensus.PubKey

	if *valKeyPath != "" {
		valKey, err := loadValidatorKey(*valKeyPath)
		if err != nil {
			log.Error("Failed to load validator key", "err", err)
			return
		}
		privVal = consensus.NewPrivValidatorLocal(valKey)
		pubVal, err = privVal.GetPubKey(rootCtx)
		if err != nil {
			log.Error("Failed to load valiator pub key", "err", err)
			return
		}
		log.Info("Running validator", "addr", pubVal.Address())
	}

	vals := make([]common.Address, len(*validatorSet))
	found := false
	for i, addrStr := range *validatorSet {
		addr := common.HexToAddress(addrStr)
		if pubVal != nil && addr == pubVal.Address() {
			found = true
		}
		vals[i] = addr
	}

	if pubVal != nil && !found {
		log.Error("Current validator is not in validator set")
		return
	} else {
		log.Info("Validators", "vals", vals)
	}

	gcs := consensus.MakeGenesisChainState("test", uint64(time.Now().UnixMilli()), vals)

	db, err := leveldb.OpenFile(*datadir, &opt.Options{ErrorIfExist: true})
	if err != nil {
		log.Error("Failed to create db", "err", err)
		return
	}

	consensusState := consensus.NewConsensusState(rootCtx, consensus.NewDefaultConsesusConfig(), *gcs, consensus.NewDefaultBlockExecutor(db), NewDefaultBlockStore(db))
	consensusState.SetPrivValidator(privVal)

	// Run block sync before consensus?
	consensusState.Start(rootCtx)

	go p2p.Run(obsvC, sendC, priv, *p2pPort, *p2pNetworkID, *p2pBootstrap, *nodeName, rootCtxCancel)

	// Running the node
	log.Info("Running the node")

	<-rootCtx.Done()
}

func getOrCreateNodeKey(path string) (p2pcrypto.PrivKey, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("No node key found, generating a new one...", "path", path)

			priv, _, err := p2pcrypto.GenerateKeyPair(p2pcrypto.Ed25519, -1)
			if err != nil {
				panic(err)
			}

			s, err := p2pcrypto.MarshalPrivateKey(priv)
			if err != nil {
				panic(err)
			}

			err = ioutil.WriteFile(path, s, 0600)
			if err != nil {
				return nil, fmt.Errorf("failed to write node key: %w", err)
			}

			return priv, nil
		} else {
			return nil, fmt.Errorf("failed to read node key: %w", err)
		}
	}

	priv, err := p2pcrypto.UnmarshalPrivateKey(b)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal node key: %w", err)
	}

	peerID, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		panic(err)
	}

	log.Info("Found existing node key",
		"path", path,
		"peerID", peerID)

	return priv, nil
}
