package gomesh

import (
	"errors"
	"sync"

	config "github.com/dynamicgo/go-config"
	"github.com/dynamicgo/xerrors"

	"github.com/dynamicgo/slf4go"

	"github.com/dynamicgo/injector"
	"google.golang.org/grpc"
)

// errors
var (
	ErrAgent = errors.New("agent implement not found")
)

// GrpcHandle .
type GrpcHandle func(server *grpc.Server) error

// GrpcService .
type GrpcService interface {
	injector.Service
	GrpcName() string
	GrpcHandle(server *grpc.Server) error
}

// GrpcServiceWithOption .
type GrpcServiceWithOption interface {
	GrpcServerOption() []grpc.ServerOption
}

// LookupF .
type LookupF func(*grpc.ClientConn) (injector.Service, error)

// Agent service mesh inprocess agent
type Agent interface {
	Startup(config config.Config) error
	Connect(serviceName string, options ...grpc.DialOption) (*grpc.ClientConn, error)
	ListenAndServe(serviceName string, handle GrpcHandle, options ...grpc.ServerOption) error
}

// RegisterAgent .
func RegisterAgent(agent Agent) {
	injector.Register("mesh.agent", agent)
}

type lookupContext struct {
	F       LookupF
	Options []grpc.DialOption
}

type lookupRegister struct {
	slf4go.Logger
	services map[string]lookupContext
}

var lookupRegisterGlobal *lookupRegister
var lookupRegisterOnce sync.Once

func lookupRegisterGlobalCreator() {
	lookupRegisterGlobal = &lookupRegister{
		Logger:   slf4go.Get("mesh-lookup"),
		services: make(map[string]lookupContext),
	}
}

// Lookup .
func Lookup(serviceName string, F LookupF, options ...grpc.DialOption) {
	lookupRegisterOnce.Do(lookupRegisterGlobalCreator)

	lookupRegisterGlobal.services[serviceName] = lookupContext{
		F:       F,
		Options: options,
	}
}

// Startup .
func Startup(config config.Config) error {
	var agent Agent
	if !injector.Get("mesh.agent", &agent) {
		return xerrors.Wrapf(ErrAgent, "call RegisterAgent first")
	}

	lookupRegisterOnce.Do(lookupRegisterGlobalCreator)

	services := lookupRegisterGlobal.services

	for name, context := range services {
		registerLookupService(name, context, agent)
	}

	return nil
}

func registerLookupService(name string, context lookupContext, agent Agent) {

	injector.RegisterService(name, func(config config.Config) (injector.Service, error) {
		conn, err := agent.Connect(name, context.Options...)

		if err != nil {
			return nil, xerrors.Wrapf(err, "agent connect to service %s error", name)
		}

		s, err := context.F(conn)

		if err != nil {
			return nil, xerrors.Wrapf(err, "create service %s agent error", name)
		}

		return s, nil
	})
}
