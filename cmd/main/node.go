package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/go-minimal-pbft/consensus"
	"github.com/go-minimal-pbft/p2p"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

var (
	p2pNetworkID *string
	p2pPort      *uint
	p2pBootstrap *string
	nodeKeyPath  *string
	nodeName     *string
	verbosity    *int
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

	go p2p.Run(obsvC, sendC, priv, *p2pPort, *p2pNetworkID, *p2pBootstrap, *nodeName, rootCtxCancel)

	consensus.NewConsensusState(rootCtx, consensus.NewDefaultConsesusConfig(), consensus.ChainState{}, consensus.NewDefaultBlockExecutor(nil), NewDefaultBlockStore(nil))

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
