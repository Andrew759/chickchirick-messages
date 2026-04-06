package factory

import (
	"net"
	"net/http"
	"time"
)

func InitHttpClient() *http.Client {
	return &http.Client{
		//Тайм-аут для глобального запроса
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				//Тайм-аут для запроса и ожидания соединения
				Timeout: time.Second,
			}).DialContext,
			//Тайм-аут для TLS рукопожатия
			TLSHandshakeTimeout: time.Second,
			//Тайм-аут для ожидания получения заголовков
			ResponseHeaderTimeout: time.Second,
		},
	}
}
