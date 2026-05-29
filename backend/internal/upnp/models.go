package upnp

import "github.com/clagon/port-mapper/backend/internal/application"

// PortMapping は、ルーターに対するポート転送の要求情報を表すドメイン型（application.PortMapping）のエイリアスです。
type PortMapping = application.PortMapping

// MaxLeaseDurationSeconds は、ポートマッピングの最大リース期間の定数エイリアスです。
const MaxLeaseDurationSeconds = application.MaxLeaseDurationSeconds

// DiscoveryResult は、探索されたルーター制御エンドポイント情報を表すドメイン型（application.DiscoveryResult）のエイリアスです。
type DiscoveryResult = application.DiscoveryResult
