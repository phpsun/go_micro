package util

import (
	"bytes"
	"common/tlog"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
)

func NewHttpClient() *http.Client {
	return &http.Client{
		Timeout: 8 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:          200,
			MaxIdleConnsPerHost:   100,
			IdleConnTimeout:       60 * time.Second,
			DisableCompression:    true,
			ResponseHeaderTimeout: 6 * time.Second,
			DialContext: (&net.Dialer{
				Timeout: 2 * time.Second,
			}).DialContext,
		},
	}
}

func NewHttpClientWithShortTimeout() *http.Client {
	return &http.Client{
		Timeout: 1200 * time.Millisecond,
		Transport: &http.Transport{
			MaxIdleConns:          200,
			MaxIdleConnsPerHost:   100,
			IdleConnTimeout:       60 * time.Second,
			DisableCompression:    true,
			ResponseHeaderTimeout: 1000 * time.Millisecond,
			DialContext: (&net.Dialer{
				Timeout: 300 * time.Millisecond,
			}).DialContext,
		},
	}
}

func HttpPost(client *http.Client, url string, headers map[string]string, params interface{}, ret interface{}) error {
	reqJSON, _ := json.Marshal(params)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
	if err != nil {
		tlog.Error(err, url)
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json;charset=UTF-8")
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		tlog.Error(err, url)
		return err
	}

	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			tlog.Error(err, url)
			return err
		}

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			switch ret.(type) {
			case *string:
				*(ret.(*string)) = string(body)
			default:
				if err := json.Unmarshal(body, ret); err != nil {
					tlog.Error(err, url)
					return err
				}
			}
			return nil
		} else {
			switch ret.(type) {
			case *string:
				*(ret.(*string)) = string(body)
			}
			errMsg := fmt.Sprintf("Http status error: %d, %s", resp.StatusCode, url)
			err = MakeGrpcError(int32(resp.StatusCode), errMsg)
			tlog.Error(err)
			return err
		}
	}
	err = fmt.Errorf("Http no body: %s", url)
	tlog.Error(err)
	return err
}

func HttpPostWithForm(client *http.Client, url string, headers map[string]string, params map[string]string, ret interface{}) error {
	var hreq http.Request
	hreq.ParseForm()
	for k, v := range params {
		hreq.Form.Add(k, v)
	}
	httpReq, err := http.NewRequest("POST", url, strings.NewReader(hreq.Form.Encode()))
	if err != nil {
		tlog.Error(err, url)
		return err
	}

	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		tlog.Error(err, url)
		return err
	}

	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			tlog.Error(err, url)
			return err
		}

		if resp.StatusCode == http.StatusOK {
			switch ret.(type) {
			case *string:
				*(ret.(*string)) = string(body)
			case *[]byte:
				*(ret.(*[]byte)) = body
			default:
				err = fmt.Errorf("Unsupport return type: %s", url)
				tlog.Error(err)
				return err
			}
			return nil
		} else {
			switch ret.(type) {
			case *string:
				*(ret.(*string)) = string(body)
			case *[]byte:
				*(ret.(*[]byte)) = body
			}
			errMsg := fmt.Sprintf("Http status error: %d, %s", resp.StatusCode, url)
			err = MakeGrpcError(int32(resp.StatusCode), errMsg)
			tlog.Error(err)
			return err
		}
	}
	err = fmt.Errorf("Http no body: %s", url)
	tlog.Error(err)
	return err
}

func HttpGet(client *http.Client, url string, headers map[string]string, ret interface{}) error {
	reqest, err := http.NewRequest("GET", url, nil)
	if err != nil {
		tlog.Error(err, url)
		return err
	}
	for k, v := range headers {
		reqest.Header.Set(k, v)
	}

	resp, err := client.Do(reqest)
	if err != nil {
		tlog.Error(err, url)
		return err
	}

	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			tlog.Error(err, url)
			return err
		}

		if resp.StatusCode == http.StatusOK {
			switch ret.(type) {
			case *string:
				*(ret.(*string)) = string(body)
			default:
				if err := json.Unmarshal(body, ret); err != nil {
					tlog.Error(err, url)
					return err
				}
			}
			return nil
		} else {
			switch ret.(type) {
			case *string:
				*(ret.(*string)) = string(body)
			}
			errMsg := fmt.Sprintf("Http status error: %d, %s", resp.StatusCode, url)
			err = MakeGrpcError(int32(resp.StatusCode), errMsg)
			tlog.Error(err)
			return err
		}
	}
	err = fmt.Errorf("Http no body: %s", url)
	tlog.Error(err)
	return err
}
