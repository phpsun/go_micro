package logic

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func GetHttpHandler() http.Handler {
	gin.SetMode(gin.ReleaseMode)
	e := gin.New()
	//中间件
	e.Use(httpPreprocess)

	apiRouter := e.Group("/api")
	{
		//conf
		confRouter := apiRouter.Group("conf")
		{
			confRouter.GET("/region-info", HandleConfInfo)
		}
	}

	return e
}

func httpPreprocess(c *gin.Context) {
	c.Next()
}
