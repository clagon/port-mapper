// Package upnp は UPnP (Universal Plug and Play) プロトコルの実装（SSDP探索、SOAP要求、ポートマッピング等）を提供します。
package upnp

import (
	"net/http"
	"time"

	"github.com/clagon/port-mapper/backend/internal/application"
)

// DiscoveryClient は、アプリケーション層の探索インターフェース（application.DiscoveryClient）にUPnP探索の実装を適合させるためのアダプター構造体です。
type DiscoveryClient struct{}

// NewDiscoveryClient は、UPnP 探索アダプターの新しいインスタンスを作成します。
func NewDiscoveryClient() DiscoveryClient {
	return DiscoveryClient{}
}

// Discover は、ネットワーク上の適合するUPnPゲートウェイを探索し、ドメイン探索結果モデルを返します。
func (DiscoveryClient) Discover() (application.DiscoveryResult, error) {
	return Discover()
}

// NewSOAPPortMapper は、UPnP 探索結果からアプリケーション層（application.PortMapper）に適合した SOAP クライアント転送操作インスタンスを生成するファクトリ関数です。
func NewSOAPPortMapper(result application.DiscoveryResult) application.PortMapper {
	return &SOAPClient{
		Endpoint:    result.ControlURL,
		ServiceType: result.ServiceType,
		HTTPClient:  &http.Client{Timeout: 5 * time.Second},
	}
}
