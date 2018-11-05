// Package basic The basic agent implement
package basic

import (
	"net"

	"github.com/dynamicgo/xerrors"

	"github.com/dynamicgo/go-config-extend"

	"github.com/dynamicgo/slf4go"

	"github.com/dynamicgo/gomesh"

	config "github.com/dynamicgo/go-config"
	"google.golang.org/grpc"
)

type agentImpl struct {
	slf4go.Logger
	config config.Config
}

func newAgent() *agentImpl {
	return &agentImpl{
		Logger: slf4go.Get("basic-agent"),
	}
}

func (agent *agentImpl) Start(config config.Config) error {
	agent.config = config

	return nil
}

func (agent *agentImpl) Config(name string) (config.Config, error) {
	config, err := extend.SubConfig(agent.config, "gomesh", "service", name)

	if err != nil {
		return nil, xerrors.Wrapf(err, "get config gomesh.service.%s error", name)
	}

	return config, nil
}

func (agent *agentImpl) Listen(name string) (net.Listener, error) {

	config, err := agent.Config(name)

	if err != nil {
		return nil, err
	}

	laddr := config.Get("laddr").String(":2018")

	listener, err := net.Listen("tcp", laddr)

	if err != nil {
		return nil, xerrors.Wrapf(err, "service %s create listener %s err", name, laddr)
	}

	return listener, nil
}

func (agent *agentImpl) Connect(name string, options ...grpc.DialOption) (*grpc.ClientConn, error) {

	config, err := agent.Config(name)

	if err != nil {
		return nil, err
	}

	remote := config.Get("remote").String("localhost:2018")

	conn, err := grpc.Dial(remote, options...)

	if err != nil {
		return nil, xerrors.Wrapf(err, "connect to rpc server %s with url %s error", name, remote)
	}

	return conn, nil
}

func init() {
	gomesh.RegisterAgent(newAgent())
}
