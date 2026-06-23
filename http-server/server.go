package httpserver

import (
	"github.com/gin-gonic/gin"
)

const (
	PORT = "8000"
)

func StartServer() {
	// set up gin server
	router := gin.Default()

	// init configs
	initConfigs()

	// router paths
	router.POST("/channel", genrateChannelId)
	router.POST("/init", initNode)

	router.Run("localhost:" + PORT)
}
