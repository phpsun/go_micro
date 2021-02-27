package project

import (
	"common/tlog"
	"common/util"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type RobotDingDing struct {
	env        string
	url        string
	secret     string
	httpClient *http.Client
}

func NewRobotDingDing(env string, app string) *RobotDingDing {
	host, _ := os.Hostname()
	r := &RobotDingDing{
		env:        "env: " + env + "\n" + "host: " + host + "\n" + "app: " + app + "\n",
		url:        "",
		secret:     "",
		httpClient: util.NewHttpClient(),
	}
	return r
}

func (this *RobotDingDing) SendMsg(msg string) string {
	timestamp := strconv.FormatInt(time.Now().Unix()*1000, 10)
	strSign := timestamp + "\n" + this.secret

	h := hmac.New(sha256.New, []byte(this.secret))
	h.Write([]byte(strSign))
	sign := url.QueryEscape(base64.StdEncoding.EncodeToString(h.Sum(nil)))
	ddurl := this.url + "&timestamp=" + timestamp + "&sign=" + sign

	type msgText struct {
		Content string `json:"content"`
	}
	type msgData struct {
		Msgtype string  `json:"msgtype"`
		Text    msgText `json:"text"`
	}

	data := msgData{Msgtype: "text", Text: msgText{Content: this.env + msg}}
	var ret string
	err := util.HttpPost(this.httpClient, ddurl, nil, data, &ret)
	if err != nil {
		tlog.Error(err)
	}
	return ret
}
