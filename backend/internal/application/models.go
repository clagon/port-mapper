package application

// PortMapping は、ルーターに対する単一のポートマッピング要求または現在のマッピング状態を表します。
type PortMapping struct {
	Protocol             string `json:"protocol"`
	ExternalPort         int    `json:"external_port"`
	InternalIP           string `json:"internal_ip"`
	InternalPort         int    `json:"internal_port"`
	Description          string `json:"description"`
	LeaseDurationSeconds int    `json:"lease_duration_seconds"`
}

// MaxLeaseDurationSeconds は、ポートマッピングの最大リース期間（7日間）を秒単位で表した定数です。
const MaxLeaseDurationSeconds = 7 * 24 * 60 * 60

// DiscoveryResult は、UPnP の探索プロセスによって発見された、選択済みのルーター制御エンドポイントの情報を表します。
type DiscoveryResult struct {
	ServiceType string
	ControlURL  string
}
