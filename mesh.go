package gomesh

import (
	"context"
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

// TccService .
type TccService interface {
	GrpcService
	TccHandle(server TccServer) error
}

// TccResource .
type TccResource struct {
	GrpcRequireFullMethod string
	Commit                func(txid string) error
	Cancel                func(txid string) error
}

// TccServer .
type TccServer interface {
	Start(config config.Config) error
	Register(tccResource TccResource) error
	NewTx(parentTxid string) (string, error)
	Commit(txid string) error
	Cancel(txid string) error
	BeforeRequire(ctx context.Context, GrpcRequireFullMethod string) error
	AfterRequire(ctx context.Context, GrpcRequireFullMethod string) error
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

// RegisterTccServer .
func RegisterTccServer(server TccServer) {
	injector.Register("mesh.tccServer", server)
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
	var tccServer TccServer

	if !injector.Get("mesh.agent", &agent) {
		return xerrors.Wrapf(ErrAgent, "must import mesh.agent implement package")
	}

	if err := agent.Start(config); err != nil {
		return err
	}

	if injector.Get("mesh.tccServer", &tccServer) {
		if err := tccServer.Start(config); err != nil {
			return err
		}
	}

	return getServiceRegister().Start(agent, tccServer)
}

// GetTccServer .
func GetTccServer() TccServer {
	var tccServer TccServer

	if injector.Get("mesh.tccServer", &tccServer) {
		err := xerrors.Wrapf(ErrAgent, "must import mesh.tcc_resource_manager implement package")
		panic(err)
	}

	return tccServer
}
