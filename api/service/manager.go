package service

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/harmony-one/harmony/internal/utils"
)

// Type is service type.
type Type byte

// Constants for Type.
const (
	ClientSupport Type = iota
	SupportExplorer
	Consensus
	BlockProposal
	NetworkInfo
	Prometheus
	Synchronize
)

func (t Type) String() string {
	switch t {
	case SupportExplorer:
		return "SupportExplorer"
	case ClientSupport:
		return "ClientSupport"
	case Consensus:
		return "Consensus"
	case BlockProposal:
		return "BlockProposal"
	case NetworkInfo:
		return "NetworkInfo"
	case Prometheus:
		return "Prometheus"
	case Synchronize:
		return "Synchronize"
	default:
		return "Unknown"
	}
}

// Service is the collection of functions any service needs to implement.
type Service interface {
	Start() error
	Stop() error
	APIs() []rpc.API // the list of RPC descriptors the service provides
}

// Manager stores all services for service manager.
type Manager struct {
	services   []Service
	serviceMap map[Type]Service
}

// NewManager creates a new manager
func NewManager() *Manager {
	return &Manager{
		services:   nil,
		serviceMap: make(map[Type]Service),
	}
}

// Register registers new service to service store.
func (m *Manager) Register(t Type, service Service) {
	utils.Logger().Info().Int("service", int(t)).Msg("Register Service")
	if _, ok := m.serviceMap[t]; ok {
		utils.Logger().Error().Int("service", int(t)).Msg("This service is already included")
		return
	}
	m.services = append(m.services, service)
	m.serviceMap[t] = service
}

// GetServices returns all registered services.
func (m *Manager) GetServices() []Service {
	return m.services
}

// StartServices run all registered services.
func (m *Manager) StartServices() {
	for _, service := range m.services {
		service.Start()
	}
}

// StopServices stops all services in the reverse order.
func (m *Manager) StopServices() {
	size := len(m.services)
	for i := size - 1; i >= 0; i-- {
		fmt.Println("stopping service", m.typeByService(m.services[i]).String(), time.Now())
		m.services[i].Stop()
	}
}

func (m *Manager) typeByService(target Service) Type {
	for t, s := range m.serviceMap {
		if s == target {
			return t
		}
	}
	return 32
}
