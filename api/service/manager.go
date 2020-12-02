package service

import (
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/harmony-one/harmony/internal/utils"
)

// ActionType is the input for Service Manager to operate.
type ActionType byte

// Constants for Action Type.
const (
	Start ActionType = iota
	Stop
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
	default:
		return "Unknown"
	}
}

// Action is type of service action.
type Action struct {
	Action      ActionType
	ServiceType Type
	Params      map[string]interface{}
}

// Interface is the collection of functions any service needs to implement.
type Interface interface {
	Specifier() string
	StartService()
	StopService()
	APIs() []rpc.API // the list of RPC descriptors the service provides
}

// Manager stores all services for service manager.
type Manager struct {
	services      map[Type]Interface
	actionChannel chan *Action
}

// GetServices returns all registered services.
func (m *Manager) GetServices() map[Type]Interface {
	return m.services
}

// Register registers new service to service store.
func (m *Manager) Register(t Type, service Interface) {
	utils.Logger().Info().Int("service", int(t)).Msg("Register Service")
	if m.services == nil {
		m.services = make(map[Type]Interface)
	}
	if _, ok := m.services[t]; ok {
		utils.Logger().Error().Int("servie", int(t)).Msg("This service is already included")
		return
	}
	m.services[t] = service
}

// SetupServiceManager inits service map and start service manager.
func (m *Manager) SetupServiceManager() {
	m.InitServiceMap()
	m.actionChannel = m.StartServiceManager()
}

// RegisterService is used for testing.
func (m *Manager) RegisterService(t Type, service Interface) {
	m.Register(t, service)
}

// InitServiceMap initializes service map.
func (m *Manager) InitServiceMap() {
	m.services = make(map[Type]Interface)
}

// TakeAction is how service manager handles the action.
func (m *Manager) TakeAction(action *Action) {
	if m.services == nil {
		utils.Logger().Error().Msg("Service store is not initialized")
		return
	}
	if service, ok := m.services[action.ServiceType]; ok {
		switch action.Action {
		case Start:
			service.StartService()
		case Stop:
			service.StopService()
		}
	}
}

// StartServiceManager starts service manager.
func (m *Manager) StartServiceManager() chan *Action {
	ch := make(chan *Action)
	go func() {
		for {
			select {
			case action := <-ch:
				m.TakeAction(action)
			}
		}
	}()
	return ch
}

// RunServices run registered services.
func (m *Manager) RunServices() {
	for serviceType := range m.services {
		action := &Action{
			Action:      Start,
			ServiceType: serviceType,
		}
		m.TakeAction(action)
	}
}

// StopService stops service with type t.
func (m *Manager) StopService(t Type) {
	if service, ok := m.services[t]; ok {
		service.StopService()
	}
}

// StopServicesByRole stops all service of the given role.
func (m *Manager) StopServicesByRole(liveServices []Type) {
	marked := make(map[Type]bool)
	for _, s := range liveServices {
		marked[s] = true
	}

	for t := range m.GetServices() {
		if _, ok := marked[t]; !ok {
			m.StopService(t)
		}
	}
}
