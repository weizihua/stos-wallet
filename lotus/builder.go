package lotus

import (
	"context"
	"fmt"
	"github.com/civet148/log"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api/apistruct"
	ma "github.com/multiformats/go-multiaddr"
	dns "github.com/multiformats/go-multiaddr-dns"
	"net/http"
	"strings"
	"time"
)

const (
	LOTUS_ENV_FULLNODE_API              = "FULLNODE_API_INFO"
	LOTUS_ENV_MINER_API                 = "MINER_API_INFO"
	DEFAULT_JSONRPC_WEBSOCKET_PREFIX    = "ws://"
	DEFAULT_JSONRPC_WEBSOCKET_RPC_V0    = "/rpc/v0"
	DEFAULT_JSONRPC_WEBSOCKET_NAMESPACE = "Filecoin"
)

type LotusBuilder func(ctx context.Context) (*apistruct.FullNodeStruct, func(), error)
type MinerBuilder func(ctx context.Context) (*apistruct.StorageMinerStruct, func(), error)

func ParseApiInfo(strApiInfo string) (maddr ma.Multiaddr, authToken string) {
	parts := strings.Split(strApiInfo, ":")
	if len(parts) < 2 {
		panic(fmt.Errorf("FullnodeApiInfo is not valid: %s", strApiInfo))
	}
	authToken = parts[0]

	addr := strings.ReplaceAll(parts[1], "/http", "")
	maddr, err := ma.NewMultiaddr(addr)
	if err != nil {
		panic(err)
	}

	return maddr, authToken
}

// NewBuilder creates a new LotusBuilder
// maddr: multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/5555")
// token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJyZWFkIiwid3JpdGUiLCJzaWduIiwiYWRtaW4iXX0.4KpuySIvV4n6kBEXQOle-hi1Ec3lyUmRYCknz4NQyLM"
func NewLotusBuilder(strAPI string, connRetries int) (LotusBuilder, error) {
	var maddr ma.Multiaddr
	var authToken string

	maddr, authToken = ParseApiInfo(strAPI)
	addr, err := TCPAddrFromMultiAddr(maddr)
	if err != nil {
		return nil, err
	}
	headers := http.Header{
		"Authorization": []string{"Bearer " + authToken},
	}

	return func(ctx context.Context) (*apistruct.FullNodeStruct, func(), error) {
		var api apistruct.FullNodeStruct
		var closer jsonrpc.ClientCloser
		var err error
		for i := 0; i < connRetries; i++ {
			if ctx.Err() != nil {
				return nil, nil, fmt.Errorf("canceled by context")
			}
			closer, err = jsonrpc.NewMergeClient(context.Background(),
				DEFAULT_JSONRPC_WEBSOCKET_PREFIX+addr+DEFAULT_JSONRPC_WEBSOCKET_RPC_V0,
				DEFAULT_JSONRPC_WEBSOCKET_NAMESPACE,
				[]interface{}{
					&api.Internal,
					&api.CommonStruct.Internal,
				}, headers)
			if err == nil {
				break
			}
			log.Warnf("connect to remote lotus error [%s], retrying...", err)
			time.Sleep(time.Second * 10)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("couldn't connect to lotus server [%s]", err)
		}

		return &api, closer, nil
	}, nil
}

// NewBuilder creates a new MinerBuilder
// maddr: multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/5555")
// token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJyZWFkIiwid3JpdGUiLCJzaWduIiwiYWRtaW4iXX0.4KpuySIvV4n6kBEXQOle-hi1Ec3lyUmRYCknz4NQyLM"
func NewMinerBuilder(strAPI string, connRetries int) (MinerBuilder, error) {
	var maddr ma.Multiaddr
	var authToken string

	maddr, authToken = ParseApiInfo(strAPI)
	addr, err := TCPAddrFromMultiAddr(maddr)
	if err != nil {
		return nil, err
	}
	headers := http.Header{
		"Authorization": []string{"Bearer " + authToken},
	}

	return func(ctx context.Context) (*apistruct.StorageMinerStruct, func(), error) {
		var api apistruct.StorageMinerStruct
		var closer jsonrpc.ClientCloser
		var err error
		for i := 0; i < connRetries; i++ {
			if ctx.Err() != nil {
				return nil, nil, fmt.Errorf("canceled by context")
			}
			closer, err = jsonrpc.NewMergeClient(context.Background(),
				DEFAULT_JSONRPC_WEBSOCKET_PREFIX+addr+DEFAULT_JSONRPC_WEBSOCKET_RPC_V0,
				DEFAULT_JSONRPC_WEBSOCKET_NAMESPACE,
				[]interface{}{
					&api.Internal,
					&api.CommonStruct.Internal,
				}, headers)
			if err == nil {
				break
			}
			log.Warnf("connect to remote miner error [%s], retrying...", err)
			time.Sleep(time.Second * 10)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("couldn't connect to miner [%s]", err)
		}

		return &api, closer, nil
	}, nil
}

// TCPAddrFromMultiAddr converts a multiaddress to a string representation of a tcp address.
func TCPAddrFromMultiAddr(maddr ma.Multiaddr) (string, error) {
	if maddr == nil {
		return "", fmt.Errorf("invalid address")
	}

	var ip string
	if _, err := maddr.ValueForProtocol(ma.P_DNS4); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		maddrs, err := dns.Resolve(ctx, maddr)
		if err != nil {
			return "", fmt.Errorf("resolving dns: %s", err)
		}
		for _, m := range maddrs {
			if ip, err = getIPFromMaddr(m); err == nil {
				break
			}
		}
	} else {
		ip, err = getIPFromMaddr(maddr)
		if err != nil {
			return "", fmt.Errorf("getting ip from maddr: %s", err)
		}
	}

	tcp, err := maddr.ValueForProtocol(ma.P_TCP)
	if err != nil {
		return "", fmt.Errorf("getting port from maddr: %s", err)
	}
	return fmt.Sprintf("%s:%s", ip, tcp), nil
}

func getIPFromMaddr(maddr ma.Multiaddr) (string, error) {
	if ip, err := maddr.ValueForProtocol(ma.P_IP4); err == nil {
		return ip, nil
	}
	if ip, err := maddr.ValueForProtocol(ma.P_IP6); err == nil {
		return fmt.Sprintf("[%s]", ip), nil
	}
	return "", fmt.Errorf("no ip in multiaddr")
}
