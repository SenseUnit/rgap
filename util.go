package rgap

import (
	"context"
	"net"
)

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

func Must[V any](value V, err error) V {
	if err != nil {
		panic(err)
	}
	return value
}
