package httpserver

import (
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/bnb-chain/tss/client"
	"github.com/bnb-chain/tss/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type KeygenRequest struct {
	Home      string `form:"home" json:"home" xml:"home" binding:"required"`
	Vault     string `form:"vault" json:"vault" xml:"vault" binding:"required"`
	Password  string `form:"password" json:"password" xml:"password" binding:"required"`
	Parties   int    `form:"parties" json:"parties" xml:"parties" binding:"required"`
	Threshold int    `form:"threshold" json:"threshold" xml:"threshold" binding:"required"`
	ChannelId string `form:"channel_id" json:"channel_id" xml:"channel_id" binding:"required"`
}

func (keygenRequest KeygenRequest) validate() error {
	if keygenRequest.Home == "" {
		return fmt.Errorf("home directory is required")
	}

	if keygenRequest.Vault == "" {
		return fmt.Errorf("vault is required")
	}

	if keygenRequest.Password == "" || len(keygenRequest.Password) < 9 {
		return fmt.Errorf("password is required and must be at least 9 characters long")
	}

	if keygenRequest.Parties <= 0 {
		return fmt.Errorf("parties must be greater than 0")
	}

	if keygenRequest.Threshold <= 0 || keygenRequest.Threshold >= keygenRequest.Parties {
		return fmt.Errorf("threshold must be greater than 0 and less than or equal to parties")
	}

	if keygenRequest.ChannelId == "" {
		return fmt.Errorf("channel id is required")
	}

	return nil
}

func keygen(ctx *gin.Context) {
	var keygenRequest KeygenRequest
	if err := ctx.ShouldBindJSON(&keygenRequest); err != nil {
		Error(ctx, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// validate request
	if err := keygenRequest.validate(); err != nil {
		Error(ctx, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// prepare for keygen
	home := fmt.Sprintf("%s/%s", HOME_DIR, keygenRequest.Home)
	initKeygenViper(home, keygenRequest.Vault, keygenRequest.Password, keygenRequest.ChannelId, keygenRequest.Parties, keygenRequest.Threshold)

	if err := common.ReadConfigFromHome(viper.GetViper(), false, home, keygenRequest.Vault, keygenRequest.Password); err != nil {
		Error(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
		return
	}

	initLogLevel(common.TssCfg)

	// run keygen
	isExist := checkKeyExist()
	if isExist {
		pubkey, address, err := GetPubkeyAndAddress(home, keygenRequest.Vault, keygenRequest.Password)
		if err != nil {
			Error(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
			return
		}

		Ok(ctx, gin.H{
			"address": address,
			"pubkey":  pubkey,
		})
		return
	}

	err := bootstrap()
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
		return
	}

	err = checkParties()
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
		return
	}

	c := client.NewTssClient(&common.TssCfg, client.KeygenMode, false)
	c.Start()

	updateConfig()

	pubkey, address, err := GetPubkeyAndAddress(home, keygenRequest.Vault, keygenRequest.Password)
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
		return
	}

	Ok(ctx, gin.H{
		"address": address,
		"pubkey":  pubkey,
	})
}

func checkKeyExist() bool {
	if _, err := os.Stat(path.Join(common.TssCfg.Home, common.TssCfg.Vault, "sk.json")); err == nil {
		return true
	}

	return false
}

func checkParties() error {
	if common.TssCfg.Parties > 0 && len(common.TssCfg.ExpectedPeers) != common.TssCfg.Parties-1 {
		return fmt.Errorf("peers are not correctly set during bootstrap")
	}

	return nil
}

func GetPubkeyAndAddress(home, vault, password string) (string, string, error) {
	pubkey, err := common.LoadEcdsaPubkey(home, vault, password)
	if err != nil {
		return "", "", err
	}

	address := client.GetEvmAddress(*pubkey)
	pubkeyBytes := crypto.FromECDSAPub(pubkey)
	pubkeyHex := hexutil.Encode(pubkeyBytes)

	return pubkeyHex, address, nil
}
