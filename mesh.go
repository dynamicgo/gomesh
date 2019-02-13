package gomesh

import (
	context "context"
	"errors"
	"net"
	"sync"

	"github.com/dynamicgo/xerrors"

	config "github.com/dynamicgo/go-config"
	"github.com/dynamicgo/injector"

	"google.golang.org/grpc"
)

//go:generate protoc --go_out=plugins=grpc,paths=source_relative:. mesh.proto

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

// TccResourceService .
type TccResourceService interface {
	GrpcService
	TccResourceHandle(TccResourceManager)
}

// TccResourceManager .
type TccResourceManager interface {
	Start(config config.Config) error
	BeforeLock(ctx context.Context, requireMethodName string) error
	AfterLock(ctx context.Context, requireMethodName string) error
	Register(requireMethodName string, commit TccResourceCommitF, cancel TccResourceCancelF)
}

// TccResourceCommitF .
type TccResourceCommitF func(txid string) error

// TccResourceCancelF .
type TccResourceCancelF func(txid string) error

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

// RegisterTccResourceManager .
func RegisterTccResourceManager(agent Agent) {
	injector.Register("mesh.tcc_resource_manager", agent)
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
	var rcManager TccResourceManager

	if !injector.Get("mesh.agent", &agent) {
		return xerrors.Wrapf(ErrAgent, "must import mesh.agent implement package")
	}

	if err := agent.Start(config); err != nil {
		return err
	}

	if injector.Get("mesh.tcc_resource_manager", &agent) {
		if err := rcManager.Start(config); err != nil {
			return err
		}
	}

	return getServiceRegister().Start(agent, rcManager)
}
