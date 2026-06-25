package httpserver

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	"github.com/bnb-chain/tss/client"
	"github.com/bnb-chain/tss/common"
	"github.com/bnb-chain/tss/p2p"
	"github.com/gin-gonic/gin"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/viper"
)

type InitRequest struct {
	Home          string `form:"home" json:"home" xml:"home" binding:"required"`
	Vault         string `form:"vault" json:"vault" xml:"vault" binding:"required"`
	Moniker       string `form:"moniker" json:"moniker" xml:"moniker" binding:"required"`
	Password      string `form:"password" json:"password" xml:"password" binding:"required"`
	ListenAddress string `form:"listen_address" json:"listen_address" xml:"listen_address" binding:"required"`
}

func (initRequest InitRequest) validate() error {
	if initRequest.Home == "" {
		return fmt.Errorf("home directory is required")
	}

	if initRequest.Vault == "" {
		return fmt.Errorf("vault is required")
	}

	if initRequest.Moniker == "" {
		return fmt.Errorf("moniker is required")
	}

	if initRequest.Password == "" || len(initRequest.Password) < 9 {
		return fmt.Errorf("password is required and must be at least 9 characters long")
	}

	if initRequest.ListenAddress == "" {
		return fmt.Errorf("listen address is required")
	}

	return nil
}

func initNode(ctx *gin.Context) {
	var initRequest InitRequest
	if err := ctx.ShouldBindJSON(&initRequest); err != nil {
		Error(ctx, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// validate request
	if err := initRequest.validate(); err != nil {
		Error(ctx, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// prepare for init node
	home := fmt.Sprintf("%s/%s", HOME_DIR, initRequest.Home)
	initNodeViper(home, initRequest.Vault, initRequest.Moniker, initRequest.Password, initRequest.ListenAddress)

	if err := makeHomeDir(home, initRequest.Vault); err != nil {
		Error(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
		return
	}

	if err := common.ReadConfigFromHome(viper.GetViper(), true, home, initRequest.Vault, initRequest.Password); err != nil {
		Error(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
		return
	}

	initLogLevel(common.TssCfg)

	// init node
	err := setP2pKey()
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
		return
	}

	err = updateConfig()
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
		return
	}

	addr, err := multiaddr.NewMultiaddr(common.TssCfg.ListenAddr)
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
		return
	}

	host, err := libp2p.New(context.Background(), libp2p.ListenAddrs(addr))
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
		return
	}

	err = host.Close()
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
		return
	}

	client.Logger.Infof("Local party has been initialized under: %s\n", path.Join(common.TssCfg.Home, common.TssCfg.Vault))

	Ok(ctx, gin.H{
		"message":        "Node has been initialized successfully",
		"id":             common.TssCfg.Id,
		"home":           common.TssCfg.Home,
		"vault":          common.TssCfg.Vault,
		"moniker":        common.TssCfg.Moniker,
		"listen_address": common.TssCfg.ListenAddr,
	})
}

func makeHomeDir(home, vault string) error {
	h := path.Join(home, vault)
	if _, err := os.Stat(h); err == nil {
		// home already exists then we override it
		if _, err := os.Stat(path.Join(h, "config.json")); err == nil {
			if err := os.Remove(path.Join(h, "config.json")); err != nil {
				return err
			}
		}
		if _, err := os.Stat(path.Join(h, "node_key")); err == nil {
			if err := os.Remove(path.Join(h, "node_key")); err != nil {
				return err
			}
		}
		if _, err := os.Stat(path.Join(h, "pk.json")); err == nil {
			if err := os.Remove(path.Join(h, "pk.json")); err != nil {
				return err
			}
		}
		if _, err := os.Stat(path.Join(h, "sk.json")); err == nil {
			if err := os.Remove(path.Join(h, "sk.json")); err != nil {
				return err
			}
		}
	} else {
		if err := os.MkdirAll(h, 0700); err != nil {
			return err
		}
	}

	return nil
}

func setP2pKey() error {
	privKey, id, err := p2p.NewP2pPrivKey()
	if err != nil {
		return err
	}

	bytes, err := crypto.MarshalPrivateKey(privKey)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(path.Join(common.TssCfg.Home, common.TssCfg.Vault, "node_key"), bytes, os.FileMode(0600)); err != nil {
		return err
	}

	common.TssCfg.Id = common.TssClientId(id.String())

	return nil
}

func updateConfig() error {
	err := common.SaveConfig(&common.TssCfg, path.Join(common.TssCfg.Home, common.TssCfg.Vault))
	if err != nil {
		return err
	}

	return nil
}
