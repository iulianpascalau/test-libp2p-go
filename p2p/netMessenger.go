package p2p

import (
	"fmt"
	"strings"

	secp "github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
)

const tcpInterface = "0.0.0.0" // bind on all interfaces

// ArgsNetMessenger defines the arguments to instantiate a network messenger wrapper struct
type ArgsNetMessenger struct {
	InitialPeerList []string
	PrivateKeyBytes []byte
}

type netMessenger struct {
	*bootstrapper
	host host.Host
}

// GeneratePrivateKeyBytes will generate a byte slice that can be used as a private key
func GeneratePrivateKeyBytes() ([]byte, error) {
	privKey, err := secp.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}

	return privKey.Serialize(), nil
}

// NewNetMessenger creates a new instance of type netMessenger
func NewNetMessenger(args ArgsNetMessenger) (*netMessenger, error) {
	privKeyBytes := args.PrivateKeyBytes
	var err error
	if len(privKeyBytes) == 0 {
		fmt.Println("provided empty private key bytes, generating a new private key")
		privKeyBytes, err = GeneratePrivateKeyBytes()
		if err != nil {
			return nil, err
		}
	}

	privateKey, err := crypto.UnmarshalSecp256k1PrivateKey(privKeyBytes)
	if err != nil {
		return nil, err
	}

	transport := libp2p.Transport(tcp.NewTCPTransport)

	// always get a free port
	address := fmt.Sprintf("/ip4/%s/tcp/0", tcpInterface)
	options := []libp2p.Option{
		libp2p.ListenAddrStrings(address),
		libp2p.Identity(privateKey),
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
		// we need to disable relay option in order to save the node's bandwidth as much as possible
		libp2p.DisableRelay(),
		libp2p.NATPortMap(),
	}
	options = append(options, transport)

	h, err := libp2p.New(options...)
	if err != nil {
		return nil, err
	}

	instance := &netMessenger{
		host: h,
	}
	instance.bootstrapper, err = newBootstrapper(h, args.InitialPeerList)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Listening on the following interfaces: %s\n", strings.Join(instance.Addresses(), ", "))

	return instance, nil
}

// Addresses returns the addresses that the current messenger was able to bind to
func (netMes *netMessenger) Addresses() []string {
	addresses := make([]string, 0)
	for _, ma := range netMes.host.Addrs() {
		addresses = append(addresses, ma.String()+"/p2p/"+netMes.ID().String())
	}

	return addresses
}

// ID returns the peer ID
func (netMes *netMessenger) ID() peer.ID {
	return netMes.host.ID()
}

// Bootstrap will start the bootstrapping process
func (netMes *netMessenger) Bootstrap() {
	netMes.bootstrapper.bootstrap()
}

// ConnectedAddresses returns all connected peer's addresses
func (netMes *netMessenger) ConnectedAddresses() []string {
	conns := make([]string, 0)
	for _, c := range netMes.h.Network().Conns() {
		conns = append(conns, c.RemoteMultiaddr().String()+"/p2p/"+c.RemotePeer().String())
	}
	return conns
}

// GetConnectedness returns the connectedness with the provided peer ID
func (netMes *netMessenger) GetConnectedness(pid peer.ID) network.Connectedness {
	return netMes.host.Network().Connectedness(pid)
}

// Close will call Close on all inner components
func (netMes *netMessenger) Close() error {
	netMes.bootstrapper.close()
	return netMes.host.Close()
}
