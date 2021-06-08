package api

import (
	"github.com/gin-gonic/gin"
)

type WalletApi interface {
	MinerWithdraw(c *gin.Context)
}
