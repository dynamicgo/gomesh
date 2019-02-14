package tcc

import (
	"context"

	"github.com/dynamicgo/xerrors"

	"github.com/dynamicgo/gomesh"
	"google.golang.org/grpc/metadata"
)

var txidkey = "gomesh_tcc_txid"

// Session .
type Session interface {
	Context() context.Context
	Commit() error
	Cancel() error
}

type sessionImpl struct {
	txid      string
	tccServer gomesh.TccServer
	ctx       context.Context
}

func (session *sessionImpl) Context() context.Context {
	return session.ctx
}

func (session *sessionImpl) Commit() error {
	return session.tccServer.Commit(session.txid)
}

func (session *sessionImpl) Cancel() error {
	return session.tccServer.Cancel(session.txid)
}

// New .
func New(ctx context.Context) (Session, error) {

	tccServer := gomesh.GetTccServer()

	if tccServer == nil {
		return nil, xerrors.New("gomesh.tccServer not register")
	}

	parentTxid, _ := Txid(ctx)

	txid, err := tccServer.NewTx(parentTxid)

	if err != nil {
		return nil, err
	}

	md := metadata.Pairs(txidkey, txid)

	session := &sessionImpl{
		txid:      txid,
		tccServer: tccServer,
		ctx:       metadata.NewOutgoingContext(ctx, md),
	}

	return session, nil
}

// Txid .
func Txid(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)

	if !ok {
		return "", false
	}

	val := md.Get(txidkey)

	if len(val) > 0 {
		return val[0], true
	}

	return "", false
}
