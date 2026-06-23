package httpserver

import (
	"fmt"

	"github.com/bnb-chain/tss/cmd"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

const (
	PORT = "8000"
)

func StartServer() {
	// set up gin server
	router := gin.Default()

	router.GET("/hello", hello)
	router.GET("/bye", bye)

	router.Run("localhost:" + PORT)
}

func hello(ctx *gin.Context) {
	cmd.Execute()

	memory := viper.GetString("log_level")

	fmt.Println("memory: ", memory)

	fmt.Println("hello")
}

func bye(ctx *gin.Context) {
	memory := viper.GetString("log_level")

	fmt.Println("memory: ", memory)

	fmt.Println("bye")
}
