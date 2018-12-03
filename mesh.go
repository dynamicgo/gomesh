package gomesh

import (
	"errors"
	"net"
	"sync"

	"github.com/dynamicgo/xerrors"

	config "github.com/dynamicgo/go-config"
	"github.com/dynamicgo/injector"

	"google.golang.org/grpc"
)

// errors
var (
	ErrAgent = errors.New("agent implement not found")
)

// Service service
type Service interface {
}

// RunnableService .
type RunnableService interface {
	Start() error
}

// GrpcService export service to vist by grpc protocol
type GrpcService interface {
	Service
	GrpcHandle(server *grpc.Server) error
}

// Agent .
type Agent interface {
	Start(config config.Config) error
	Config(name string) (config.Config, error)
	Listen() (net.Listener, error)
	Connect(name string, options ...grpc.DialOption) (*grpc.ClientConn, error)
}

// RegisterAgent .
func RegisterAgent(agent Agent) {
	injector.Register("mesh.agent", agent)
}

var globalRegister Register
var once sync.Once

func getServiceRegister() Register {
	once.Do(func() {
		globalRegister = NewServiceRegister()
	})

	return globalRegister
}

// LocalService register local service
func LocalService(name string, F F) {
	getServiceRegister().LocalService(name, F)
}

// RemoteService register remote service
func RemoteService(name string, F RemoteF) {
	getServiceRegister().RemoteService(name, F)
}

// Start start gomesh
func Start(config config.Config) error {
	var agent Agent

	if !injector.Get("mesh.agent", &agent) {
		return xerrors.Wrapf(ErrAgent, "must import mesh.agent implement package")
	}

	if err := agent.Start(config); err != nil {
		return err
	}

	return getServiceRegister().Start(agent)
}
