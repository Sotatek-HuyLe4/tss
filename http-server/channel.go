package httpserver

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/bnb-chain/tss/common"
	"github.com/gin-gonic/gin"
)

type ChannelRequest struct {
	Expire int `form:"expire" json:"expire" xml:"expire" binding:"required"`
}

func genrateChannelId(ctx *gin.Context) {
	var channelRequest ChannelRequest
	if err := ctx.ShouldBindJSON(&channelRequest); err != nil {
		Error(ctx, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// validate expire time
	if channelRequest.Expire <= 0 {
		Error(ctx, http.StatusBadRequest, "INVALID_REQUEST", "expire time should be greater than 0")
		return
	}

	id, err := rand.Int(rand.Reader, big.NewInt(999))
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
		return
	}

	expireTime := time.Now().Add(time.Duration(channelRequest.Expire) * time.Minute).Unix()
	channelId := fmt.Sprintf("%.3d%s", id.Int64(), common.ConvertTimestampToHex(expireTime))

	// send success response
	Ok(ctx, gin.H{
		"channel_id": channelId,
	})
}
