package httpserver

import (
	"github.com/bnb-chain/tss/common"
	"github.com/ipfs/go-log"
	"github.com/spf13/viper"
)

func initConfigs() {
	// bind p2p configs
	viper.Set("p2p.listen", "")
	viper.Set("p2p.new_listen", "")
	viper.Set("p2p.peer_addrs", []string{})
	viper.Set("p2p.new_peer_addrs", []string{})
	viper.Set("p2p.broadcast_sanity_check", true)

	// bind kdf configs
	viper.Set("kdf.memory", 65536)
	viper.Set("kdf.iterations", 13)
	viper.Set("kdf.parallelism", 4)
	viper.Set("kdf.salt_length", 16)
	viper.Set("kdf.key_length", 48)

	// bind client configs
	viper.Set("moniker", "")
	viper.Set("vault_name", "")
	viper.Set("threshold", 0)
	viper.Set("parties", 0)
	viper.Set("new_threshold", 0)
	viper.Set("new_parties", 0)
	viper.Set("password", "")
	viper.Set("message", "")
	viper.Set("log_level", "info")
	viper.Set("channel_id", "")
	viper.Set("channel_password", "")
	viper.Set("channel_expire", 0)
	viper.Set("is_old", false)
	viper.Set("is_new_member", false)
	viper.Set("pubkey", "")

	// bind sign configs
	viper.Set("rpc_url", "http://localhost:8545")
	viper.Set("to_address", "")
	viper.Set("amount", "1")
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
