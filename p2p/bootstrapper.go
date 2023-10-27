package p2p

import (
	"context"
	"fmt"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

const protocolID = "/erd/kad/9.9.9"
const routingTableRefreshInterval = time.Second * 300
const retryReconnectionToInitialPeerListInterval = time.Second * 5
const timeToConnect = time.Second * 2

type bootstrapper struct {
	h               host.Host
	initialPeerList []string
	kadDHT          *dht.IpfsDHT
	cancelFunc      func()
	chStart         chan struct{}
}

func newBootstrapper(h host.Host, initialPeerList []string) (*bootstrapper, error) {
	instance := &bootstrapper{
		h:               h,
		chStart:         make(chan struct{}),
		initialPeerList: initialPeerList,
	}

	var ctx context.Context
	ctx, instance.cancelFunc = context.WithCancel(context.Background())

	var err error
	instance.kadDHT, err = dht.New(
		ctx,
		h,
		dht.ProtocolPrefix(protocolID),
		dht.RoutingTableRefreshPeriod(routingTableRefreshInterval),
		dht.Mode(dht.ModeServer),
	)
	if err != nil {
		return nil, err
	}
	go instance.tryStart(ctx)

	return instance, nil
}

func (b *bootstrapper) tryStart(ctx context.Context) {
	defer func() {
		err := b.kadDHT.Bootstrap(ctx)
		if err != nil {
			fmt.Printf("ERROR %s for kadDHT Bootstrap call: %s\n", err.Error(), b.h.ID().String())
		}
	}()

	if len(b.initialPeerList) == 0 {
		fmt.Printf("DEBUG: %s no initial peer list provided\n", b.h.ID().String())
		return
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("INFO: %s bootstrapper try start ended early\n", b.h.ID().String())
			return
		case <-b.chStart:
			err := b.connectToInitialPeerList()
			if err == nil {
				fmt.Printf("INFO %s CONNECTED to the network\n", b.h.ID().String())
				return
			}

			time.Sleep(retryReconnectionToInitialPeerListInterval)
		}
	}
}

// connectToInitialPeerList will return the error if there sunt
func (b *bootstrapper) connectToInitialPeerList() error {
	for _, address := range b.initialPeerList {
		err := b.connectToHost(address)
		if err != nil {
			fmt.Printf("ERROR %s: while attempting initial connection to %s: %s\n",
				b.h.ID().String(), address, err.Error())
			continue
		}

		return nil
	}

	return fmt.Errorf("unable to connect to initial peers")
}

func (b *bootstrapper) connectToHost(address string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeToConnect)
	defer cancel()

	multiAddr, err := multiaddr.NewMultiaddr(address)
	if err != nil {
		return err
	}

	pi, err := peer.AddrInfoFromP2pAddr(multiAddr)
	if err != nil {
		return err
	}

	return b.h.Connect(ctx, *pi)
}

func (b *bootstrapper) bootstrap() {
	// try a non-blocking write
	select {
	case b.chStart <- struct{}{}:
	default:
	}
}

func (b *bootstrapper) close() {
	b.cancelFunc()
}
