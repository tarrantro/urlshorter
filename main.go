package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var zapLogger *zap.Logger = newLogger()

func Getenv(env string, defaultEnv string) string {
	e := os.Getenv(env)
	if e == "" {
		e = defaultEnv
	}
	return e
}

func serve() {
	router := gin.New()

	// Add a ginzap middleware
	router.Use(ginzap.GinzapWithConfig(zapLogger, &ginzap.Config{
		TimeFormat: time.RFC3339,
		UTC: false,
		SkipPaths: []string{"/ping", "/node"},
	  }))
	// Logs all panic to error log
	//   - stack means whether output the stack info.
	router.Use(ginzap.RecoveryWithZap(zapLogger, true))
	// Wait 1 second for headless DNS sync up
	time.Sleep(1*time.Second)
	node, err := node()
	if err != nil {
		panic(fmt.Sprintf("Fail to init node, err: %s", err.Error()))
	}
	dbclient := DynamoClient()
	if dbclient == nil {
		panic(fmt.Sprintf("Fail to init db client"))
	}

	root := router.Group("/")
	root.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, "pong")
	})
	root.GET("/node", getID(node))
	root.POST("/newurl", setURL(dbclient, node))
	root.GET("/:regex", proxy(dbclient))
	router.Run()
}

func main() {
	serve()
}
