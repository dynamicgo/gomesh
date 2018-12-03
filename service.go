package gomesh

import (
	"context"
	"sync"

	"github.com/dynamicgo/xerrors/apierr"

	"google.golang.org/grpc"

	config "github.com/dynamicgo/go-config"
	"github.com/dynamicgo/injector"
	"github.com/dynamicgo/slf4go"
	"github.com/dynamicgo/xerrors"
)

// F service factory
type F func(config config.Config) (Service, error)

// RemoteF .
type RemoteF func(conn *grpc.ClientConn) (Service, error)

// Register .
type Register interface {
	LocalService(name string, F F)
	RemoteService(name string, F RemoteF)
	Start(agent Agent) error
}

type serviceF struct {
	F    F
	Name string
}

type remoteServiceF struct {
	F    RemoteF
	Name string
}

type serviceRegister struct {
	sync.RWMutex                      // mixin mutex
	slf4go.Logger                     // mixin logger
	factories       []*serviceF       // service factories
	remoteServices  []*remoteServiceF // remote services
	context         injector.Injector // inject context
	grpcServers     []*grpc.Server    // grpc server
	grpcServerNames []string          // grpc server names
	runnables       []RunnableService // runnable services
	runnableNames   []string          // runnable service names
}

// NewServiceRegister .
func NewServiceRegister() Register {
	return &serviceRegister{
		Logger:  slf4go.Get("mesh-service"),
		context: injector.New(),
	}
}

func (register *serviceRegister) checkServiceName(name string) {
	for _, serviceF := range register.factories {
		if serviceF.Name == name {
			err := xerrors.Wrapf(injector.ErrExists, "service %s exists", name)
			panic(err)
		}
	}

	for _, n := range register.remoteServices {
		if n.Name == name {
			err := xerrors.Wrapf(injector.ErrExists, "service %s exists", name)
			panic(err)
		}
	}
}

func (register *serviceRegister) LocalService(name string, F F) {

	register.Lock()
	defer register.Unlock()

	register.checkServiceName(name)

	f := &serviceF{
		Name: name,
		F:    F,
	}

	register.factories = append(register.factories, f)
}

func (register *serviceRegister) RemoteService(name string, F RemoteF) {
	register.Lock()
	defer register.Unlock()

	register.checkServiceName(name)

	register.remoteServices = append(register.remoteServices, &remoteServiceF{
		Name: name,
		F:    F,
	})
}

func (register *serviceRegister) bindRemoteServices(agent Agent) error {

	for _, sf := range register.remoteServices {
		register.InfoF("register remote service %s", sf.Name)
		conn, err := agent.Connect(sf.Name)

		if err != nil {
			return xerrors.Wrapf(err, "create remote service %s connect error", sf.Name)
		}

		service, err := sf.F(conn)

		if err != nil {
			return xerrors.Wrapf(err, "create remote service %s proxy error", sf.Name)
		}

		register.context.Register(sf.Name, service)
	}

	return nil
}

func (register *serviceRegister) Start(agent Agent) error {
	register.RLock()
	defer register.RUnlock()

	if err := register.bindRemoteServices(agent); err != nil {
		return err
	}

	// bind remote services

	var services []Service
	var serviceNames []string
	var runnables []RunnableService
	var runnableNames []string

	var grpcServices []GrpcService
	var grpcServiceNames []string

	// create services
	for _, f := range register.factories {
		register.InfoF("create service %s", f.Name)

		subconfig, err := agent.Config(f.Name)

		if err != nil {
			return xerrors.Wrapf(err, "load service %s config err", f.Name)
		}

		service, err := f.F(subconfig)

		if err != nil {
			return xerrors.Wrapf(err, "create service %s error", f.Name)
		}

		services = append(services, service)
		register.context.Register(f.Name, service)

		serviceNames = append(serviceNames, f.Name)

		if runnable, ok := service.(RunnableService); ok {
			runnables = append(runnables, runnable)
			runnableNames = append(runnableNames, f.Name)
		}

		if grpcService, ok := service.(GrpcService); ok {
			grpcServices = append(grpcServices, grpcService)
			grpcServiceNames = append(grpcServiceNames, f.Name)
		}

	}

	for i, service := range services {
		if err := register.context.Bind(service); err != nil {
			return xerrors.Wrapf(err, "service %s bind error", register.factories[i].Name)
		}
	}

	if err := register.startRunnableServices(agent, runnableNames, runnables); err != nil {
		return err
	}

	if err := register.startGrpcServices(agent, grpcServiceNames, grpcServices); err != nil {
		return err
	}

	return nil
}

func (register *serviceRegister) startRunnableServices(agent Agent, runnableNames []string, runnables []RunnableService) error {
	register.runnableNames = runnableNames
	register.runnables = runnables

	for i, runnable := range runnables {

		if err := runnable.Start(); err != nil {
			return xerrors.Wrapf(err, "start service %s error", runnableNames[i])
		}
	}

	return nil
}

func (register *serviceRegister) startGrpcServices(agent Agent, grpcServiceNames []string, grpcServices []GrpcService) error {
	register.grpcServers = nil
	register.grpcServerNames = grpcServiceNames

	listener, err := agent.Listen()

	if err != nil {
		return xerrors.Wrapf(err, "create grpc service listener error")
	}

	var server *grpc.Server

	server = grpc.NewServer(grpc.UnaryInterceptor(register.UnaryServerInterceptor))

	for i, grpcService := range grpcServices {

		if err := grpcService.GrpcHandle(server); err != nil {
			return xerrors.Wrapf(err, "call grpc service %s handle error", grpcServiceNames[i])
		}

		register.grpcServers = append(register.grpcServers, server)
	}

	go func() {
		if err := server.Serve(listener); err != nil {
			register.ErrorF("grpc serve err %s", err)
		}
	}()

	return nil
}

func (register *serviceRegister) UnaryServerInterceptor(
	ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	resp, err := handler(ctx, req)

	err = apierr.AsGrpcError(apierr.As(err, apierr.New(-1, "UNKNOWN")))

	return resp, err
}
