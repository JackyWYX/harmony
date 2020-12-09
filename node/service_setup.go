package node

import (
	msg_pb "github.com/harmony-one/harmony/api/proto/message"
	"github.com/harmony-one/harmony/api/service"
	"github.com/harmony-one/harmony/api/service/blockproposal"
	"github.com/harmony-one/harmony/api/service/consensus"
	"github.com/harmony-one/harmony/api/service/explorer"
	nodeconfig "github.com/harmony-one/harmony/internal/configs/node"
)

func (node *Node) setupForValidator() {
	// Register consensus service.
	node.serviceManager.Register(
		service.Consensus,
		consensus.New(node.BlockChannel, node.Consensus, node.startConsensus),
	)
	// Register new block service.
	node.serviceManager.Register(
		service.BlockProposal,
		blockproposal.New(node.Consensus.ReadySignal, node.WaitForConsensusReadyV2),
	)
}

func (node *Node) setupForExplorerNode() {
	node.initNodeConfiguration()

	// Register explorer service.
	node.serviceManager.Register(
		service.SupportExplorer, explorer.New(&node.SelfPeer, node.stateSync, node.Blockchain()),
	)
}

// ServiceManagerSetup setups service store.
func (node *Node) ServiceManagerSetup() {
	node.serviceManager = &service.Manager{}
	node.serviceMessageChan = make(map[service.Type]chan *msg_pb.Message)
	switch node.NodeConfig.Role() {
	case nodeconfig.Validator:
		node.setupForValidator()
	case nodeconfig.ExplorerNode:
		node.setupForExplorerNode()
	}
}

// RunServices runs registered services.
func (node *Node) RunServices() {
	node.serviceManager.StartServices()
}

// StopServices runs registered services.
func (node *Node) StopServices() {
	node.serviceManager.StopServices()
}
