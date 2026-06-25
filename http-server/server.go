package httpserver

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	HOME_DIR = ".tss"
)

func StartServer(port int) {
	// set up gin server
	router := gin.Default()

	// router paths
	router.POST("/channel", genrateChannelId)
	router.POST("/init", initNode)
	router.POST("/keygen", keygen)
	router.POST("/sign", sign)

	router.Run("localhost:" + strconv.Itoa(port))
}
