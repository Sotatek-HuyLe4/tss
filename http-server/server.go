package httpserver

import (
	"github.com/gin-gonic/gin"
)

const (
	HOME_DIR = ".tss"
	PORT     = "8000"
)

func StartServer() {
	// set up gin server
	router := gin.Default()

	// init common viper
	initCommonViper()

	// router paths
	router.POST("/channel", genrateChannelId)
	router.POST("/init", initNode)
	router.POST("/keygen", keygen)

	router.Run("localhost:" + PORT)
}
