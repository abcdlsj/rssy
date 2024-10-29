package main

import (
	"fmt"

	"github.com/abcdlsj/rssy/internal"
	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func main() {
	r := internal.ServerRouter()

	log.Infof("Running on %s", internal.SiteURL)
	r.Run(fmt.Sprintf(":%s", internal.Port))
}
