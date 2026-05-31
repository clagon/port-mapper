// Package domain は Porto のドメインモデル、境界インターフェース、およびビジネス検証ロジックを定義します。
package domain

import "errors"

// ErrNoGateway は、UPnP ゲートウェイ（ルーター）が探索プロセスで発見できなかったことを表すエラーです。
var ErrNoGateway = errors.New("no UPnP gateway discovered")

// ErrNoPortMappingEntry は、指定インデックス以降にルーター上のポートマッピングが存在しないことを表すエラーです。
var ErrNoPortMappingEntry = errors.New("no port mapping entry")

// DiscoveryClient は、ポートマッピング要求を送信するためのルーター制御エンドポイントをネットワーク上から自動検出するためのインターフェースです。
type DiscoveryClient interface {
	// Discover は、ネットワーク上のUPnPデバイスを探索し、適合するルーターのエンドポイント情報を返します。
	Discover() (DiscoveryResult, error)
}

// PortMapper は、発見されたルーターに対してポートマッピングの取得、追加、削除などのUPnP/SOAP操作を実行するためのインターフェースです。
type PortMapper interface {
	// GetExternalIPAddress は、ルーターのWAN側（外部）グローバルIPアドレスを取得します。
	GetExternalIPAddress() (string, error)
	// AddPortMapping は、ルーターに対して新しいポート転送ルールを追加します。
	AddPortMapping(PortMapping) error
	// DeletePortMapping は、指定されたプロトコルと外部ポートに対応するポート転送ルールをルーターから削除します。
	DeletePortMapping(protocol string, externalPort int) error
	// GetGenericPortMappingEntry は、ルーター上に現在登録されているポートマッピングを指定インデックスに基づいて取得します（同期用）。
	GetGenericPortMappingEntry(index int) (PortMapping, error)
}

// PortMapperFactory は、DiscoveryResult（探索結果）から具体的な PortMapper インスタンスを生成するためのファクトリ関数型です。
type PortMapperFactory func(DiscoveryResult) PortMapper
