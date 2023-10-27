package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"
	"v1/display"
	"v1/p2p"

	"github.com/urfave/cli"
)

type netMessenger interface {
	ConnectedAddresses() []string
	Addresses() []string
}

var (
	privateKeyFlag = cli.StringFlag{
		Name: "private-key",
		Usage: "optional parameter: the p2p private key as hex string used by the p2p host. Not providing this flag will autogenerate a private key, internally" +
			"Example: d4325db4db622f207ae9c72cb5b5429e87c5d06247f22dd16a8d8995770c0c89 will generate the pid 16Uiu2HAkw5dh6RoEaPHNBWdzw1ZTfNTXAB34mG3xrd8JNUN2o6S8",
		Value: "",
	}
	initialPeerList = cli.StringFlag{
		Name: "initial-peer-list",
		Usage: "optional parameter: the initial peer list that the node will attempt to connect to. They should be separated by comma." +
			"Example: /ip4/192.168.169.161/tcp/35153/p2p/16Uiu2HAkw5dh6RoEaPHNBWdzw1ZTfNTXAB34mG3xrd8JNUN2o6S8,/ip4/127.0.0.1/tcp/35153/p2p/16Uiu2HAkw5dh6RoEaPHNBWdzw1ZTfNTXAB34mG3xrd8JNUN2o6S8",
		Value: "",
	}
)

func main() {
	app := cli.NewApp()
	app.Name = "Test node CLI App"
	app.Flags = []cli.Flag{
		privateKeyFlag,
		initialPeerList,
	}
	app.Version = "v0.0.0"

	app.Action = func(c *cli.Context) error {
		return startNode(c)
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		os.Exit(1)
	}
}

func startNode(ctx *cli.Context) error {
	var privateKeyBytes []byte
	var err error

	privateKeyHex := ctx.GlobalString(privateKeyFlag.Name)
	if len(privateKeyHex) > 0 {
		privateKeyBytes, err = hex.DecodeString(privateKeyHex)
		if err != nil {
			return fmt.Errorf("%w while parsing provided private key hex string", err)
		}
	}

	var initialPeers []string
	providedInitialPeerList := ctx.GlobalString(initialPeerList.Name)
	if len(providedInitialPeerList) > 0 {
		initialPeers = strings.Split(providedInitialPeerList, ",")
	}

	argsSeeder := p2p.ArgsNetMessenger{
		InitialPeerList: initialPeers,
		PrivateKeyBytes: privateKeyBytes,
	}

	netMes, err := p2p.NewNetMessenger(argsSeeder)
	if err != nil {
		return err
	}

	netMes.Bootstrap()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-sigs:
			fmt.Println("terminating at user's signal...")
			return nil
		case <-time.After(time.Second * 5):
			displayNetMessengerInfo(netMes)
		}
	}
}

func displayNetMessengerInfo(netMes netMessenger) {
	headerSeedAddresses := []string{"Node's own addresses:"}
	addresses := make([]*display.LineData, 0)

	for _, address := range netMes.Addresses() {
		addresses = append(addresses, display.NewLineData(false, []string{address}))
	}

	tbl, _ := display.CreateTableString(headerSeedAddresses, addresses)
	fmt.Printf("\n%s\n", tbl)

	mesConnectedAddrs := netMes.ConnectedAddresses()
	sort.Slice(mesConnectedAddrs, func(i, j int) bool {
		return strings.Compare(mesConnectedAddrs[i], mesConnectedAddrs[j]) < 0
	})

	headerConnectedAddresses := []string{fmt.Sprintf("Node is connected to %d peers:", len(mesConnectedAddrs))}
	connAddresses := make([]*display.LineData, len(mesConnectedAddrs))

	for idx, address := range mesConnectedAddrs {
		connAddresses[idx] = display.NewLineData(false, []string{address})
	}

	tbl2, _ := display.CreateTableString(headerConnectedAddresses, connAddresses)
	fmt.Printf("\n%s\n", tbl2)
}
