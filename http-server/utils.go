package httpserver

import (
	"github.com/bnb-chain/tss/common"
	"github.com/ipfs/go-log"
	"github.com/spf13/viper"
)

func initCommonViper() {
	viper.Set("vault_name", "")
	viper.Set("password", "")
	viper.Set("log_level", "info")
}

func initNodeViper(home, vault, moniker, password, listenAddress string) {
	viper.Set("home", home)
	viper.Set("vault_name", vault)
	viper.Set("moniker", moniker)
	viper.Set("password", password)
	viper.Set("p2p.listen", listenAddress)

	// bind kdf configs
	viper.Set("kdf.memory", 65536)
	viper.Set("kdf.iterations", 13)
	viper.Set("kdf.parallelism", 4)
	viper.Set("kdf.salt_length", 16)
	viper.Set("kdf.key_length", 48)
}

func initKeygenViper(home, vault, password, channelId string, parties, threshold int) {
	viper.Set("home", home)
	viper.Set("vault_name", vault)
	viper.Set("password", password)
	viper.Set("parties", parties)
	viper.Set("threshold", threshold)
	viper.Set("channel_id", channelId)
	viper.Set("channel_password", password)

	viper.Set("p2p.peer_addrs", []string{})
	viper.Set("p2p.broadcast_sanity_check", true)
}

func initLogLevel(cfg common.TssConfig) {
	log.SetLogLevel("tss", cfg.LogLevel)
	log.SetLogLevel("tss-lib", cfg.LogLevel)
	log.SetLogLevel("srv", cfg.LogLevel)
	log.SetLogLevel("trans", cfg.LogLevel)
	log.SetLogLevel("p2p_utils", cfg.LogLevel)
	log.SetLogLevel("common", cfg.LogLevel)

	// libp2p loggers
	log.SetLogLevel("dht", "error")
	log.SetLogLevel("discovery", "error")
	log.SetLogLevel("swarm2", "error")
	log.SetLogLevel("stream-upgrader", "error")
}
