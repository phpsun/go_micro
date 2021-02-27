package project

import (
	"common/defs"
	"common/util"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/proto"
	"io"
	"net/http"
)

type httpRenderJson struct {
	data string
}

var jsonContentType = []string{"application/json; charset=utf-8"}

func (r httpRenderJson) Render(w http.ResponseWriter) error {
	r.WriteContentType(w)
	_, err := io.WriteString(w, r.data)
	return err
}

func (r httpRenderJson) WriteContentType(w http.ResponseWriter) {
	header := w.Header()
	header["Content-Type"] = jsonContentType
}

func OutputFullJsonResult(c *gin.Context, m *util.JsonMarshaler, msg proto.Message) {
	str, err := m.MarshalToString(msg)
	if err != nil {
		c.JSON(http.StatusOK, Fail(defs.ErrUnknownResponse, err.Error()))
		return
	}

	str = "{\"code\":0,\"message\":\"success\",\"data\":" + str + "}"
	c.Render(http.StatusOK, httpRenderJson{data: str})
}

func OutputStringResult(c *gin.Context, str string) {
	str = "{\"code\":0,\"message\":\"success\",\"data\":" + str + "}"
	c.Render(http.StatusOK, httpRenderJson{data: str})
}
