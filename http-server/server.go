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

	// router paths
	router.POST("/channel", genrateChannelId)

	router.Run("localhost:" + PORT)
}
