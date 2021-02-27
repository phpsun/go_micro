package logic

import (
	"common/defs"
	"common/discovery"
	"common/project"
	"common/proto/config"
	"common/tlog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func HandleConfInfo(c *gin.Context) {
	var err error
	req := &config.InfoReq{}
	id := c.Query("id")

	req.Id, err = strconv.ParseInt(id, 10, 64)
	if err != nil {
		tlog.Error(err)
		c.JSON(http.StatusOK, project.Fail(defs.ErrCommon, "grpc error:"+err.Error()))
		return
	}

	ctx, cancel := discovery.ContextWithStandard()
	defer cancel()
	resp, err := ThisServer.ConfigGrpc.Info(ctx, req)

	if err != nil {
		tlog.Error(err)
		ThisServer.Output.OutputGrpcError(c, err)
	} else {
		ThisServer.Output.OutputSuccess(c, resp)
	}
}
