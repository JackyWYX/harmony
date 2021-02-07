package node

import (
	"fmt"

	"github.com/harmony-one/harmony/api/service"
	"github.com/harmony-one/harmony/api/service/blockproposal"
	"github.com/harmony-one/harmony/api/service/consensus"
	"github.com/harmony-one/harmony/api/service/explorer"
	"github.com/harmony-one/harmony/internal/utils"
)

// RegisterValidatorServices register the validator services.
func (node *Node) RegisterValidatorServices() {
	if node.serviceManager == nil {
		node.serviceManager = service.NewManager()
	}

	// Register consensus service.
	node.serviceManager.Register(
		service.Consensus,
		consensus.New(node.BlockChannel, node.Consensus, node.startConsensus),
	)
	// Register new block service.
	node.serviceManager.Register(
		service.BlockProposal,
		blockproposal.New(node.Consensus.ReadySignal, node.Consensus.CommitSigChannel, node.WaitForConsensusReadyV2),
	)
}

// RegisterExplorerServices register the explorer services
func (node *Node) RegisterExplorerServices() {
	if node.serviceManager == nil {
		node.serviceManager = service.NewManager()
	}

	// Register explorer service.
	node.serviceManager.Register(
		service.SupportExplorer, explorer.New(&node.SelfPeer, node.stateSync, node.Blockchain()),
	)
}

// RegisterService register a service to the node service manager
func (node *Node) RegisterService(st service.Type, s service.Service) {
	node.serviceManager.Register(st, s)
}

// StartServices runs registered services.
func (node *Node) StartServices() {
	if node.serviceManager == nil {
		utils.Logger().Info().Msg("Service manager is not set up yet.")
		return
	}
	node.serviceManager.StartServices()
}

// StopServices runs registered services.
func (node *Node) StopServices() {
	if node.serviceManager == nil {
		utils.Logger().Info().Msg("Service manager is not set up yet.")
		return
	}
	node.serviceManager.StopServices()
}

func (node *Node) networkInfoDHTPath() string {
	return fmt.Sprintf(".dht-%s-%s-c%s",
		node.SelfPeer.IP,
		node.SelfPeer.Port,
		node.chainConfig.ChainID,
	)
}
