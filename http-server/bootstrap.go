package httpserver

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/bnb-chain/tss/client"
	"github.com/bnb-chain/tss/cmd"
	"github.com/bnb-chain/tss/common"
)

func bootstrap() error {
	src, err := common.ConvertMultiAddrStrToNormalAddr(common.TssCfg.ListenAddr)
	if err != nil {
		return err
	}
	listenAddrs := cmd.GetListenAddrs(common.TssCfg.ListenAddr)

	client.Logger.Info("waiting peers startup...")

	numOfPeers := common.TssCfg.Parties - 1
	if common.TssCfg.BMode == common.PreRegroupMode {
		numOfPeers = common.TssCfg.Threshold + common.TssCfg.NewParties
	}

	bootstrapper := common.NewBootstrapper(numOfPeers, &common.TssCfg)
	listener, err := net.Listen("tcp", src)
	client.Logger.Infof("listening on %s", src)
	if err != nil {
		return err
	}
	defer func() {
		err = listener.Close()
		if err != nil {
			client.Logger.Error(err)
		}

		client.Logger.Info("closed ssdp listener")
	}()

	done := make(chan bool)
	go cmd.AcceptConnRoutine(listener, bootstrapper, done)

	peerAddrs := cmd.FindPeerAddrsViaSsdp(numOfPeers, listenAddrs)
	client.Logger.Infof("Found peers via ssdp: %v", peerAddrs)

	go func() {
		for _, peerAddr := range peerAddrs {
			go func(peerAddr string) {
				dest, err := common.ConvertMultiAddrStrToNormalAddr(peerAddr)
				if err != nil {
					common.Panic(fmt.Errorf("failed to convert peer multiAddr to addr: %v", err))
				}

				client.Logger.Debugf("going to dial: %s", peerAddr)
				conn, err := net.Dial("tcp", dest)
				for conn == nil {
					if err != nil {
						if !strings.Contains(err.Error(), "connection refused") {
							client.Logger.Errorf("dial failed: %v", err)
							common.Panic(err)
						}
					}
					time.Sleep(time.Second)
					conn, err = net.Dial("tcp", dest)
				}

				client.Logger.Debugf("done dial: %s", peerAddr)
				defer conn.Close()

				cmd.HandleConnection(conn, bootstrapper)
			}(peerAddr)
		}

		cmd.CheckReceivedPeerInfos(bootstrapper, done)
	}()

	<-done
	err = cmd.UpdateConfigWithPeerInfos(bootstrapper)
	if err != nil {
		return err
	}

	return nil
}
