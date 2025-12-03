package web

import (
	"net"
	"net/http"
	"time"

	"go-transfer/internal/constants"
)

// CreateUploadClient 创建用于文件上传的HTTP客户端（客户端模式使用）
func CreateUploadClient() *http.Client {
	return &http.Client{
		Timeout: constants.DefaultTimeout,
		Transport: &http.Transport{
			MaxConnsPerHost:       constants.MaxConnsPerHost,
			MaxIdleConnsPerHost:   constants.MaxIdleConnsPerHost,
			MaxIdleConns:          constants.MaxIdleConns,
			IdleConnTimeout:       constants.IdleConnTimeout,
			DisableKeepAlives:     false,
			ForceAttemptHTTP2:     false, // 强制 HTTP/1.1
			ResponseHeaderTimeout: constants.ResponseTimeout,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
	}
}

// CreateForwardClient 创建用于转发的HTTP客户端（转发模式使用）
func CreateForwardClient() *http.Client {
	return &http.Client{
		Timeout: constants.DefaultTimeout,
		Transport: &http.Transport{
			DisableCompression:  true,
			DisableKeepAlives:   false,
			IdleConnTimeout:     constants.IdleConnTimeout,
			WriteBufferSize:     constants.MediumBufferSize,
			ReadBufferSize:      constants.MediumBufferSize,
			MaxIdleConns:        10,
			MaxConnsPerHost:     10,
		},
	}
}