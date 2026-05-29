package server

import (
	"github.com/clagon/port-mapper/backend/internal/application"
	"github.com/clagon/port-mapper/backend/internal/service"
)

// HealthResponse は、GET /api/health によって返されるヘルスチェック用の JSON レスポンスデータ構造です。
type HealthResponse struct {
	Ok bool `json:"ok"`
}

// ActionResponse は、更新や削除などの副作用を伴う API 呼び出しが成功した際に返される標準的な JSON レスポンスデータ構造です。
type ActionResponse struct {
	Ok bool `json:"ok"`
}

// PortMappingRequest は、POST /api/ports/open でポートを新しく開く際にクライアントから送信される JSON リクエストデータ構造です。
type PortMappingRequest struct {
	Protocol             string `json:"protocol"`
	ExternalPort         int    `json:"external_port"`
	InternalIP           string `json:"internal_ip"`
	InternalPort         int    `json:"internal_port"`
	Description          string `json:"description"`
	LeaseDurationSeconds int    `json:"lease_duration_seconds"`
}

// ClosePortRequest は、POST /api/ports/close で特定の転送ポートを閉じる際にクライアントから送信される JSON リクエストデータ構造です。
type ClosePortRequest struct {
	Protocol     string `json:"protocol"`
	ExternalPort int    `json:"external_port"`
}

// PortMappingResponse は、ポートマッピング情報の詳細をクライアントに返すための JSON データ構造です。
type PortMappingResponse struct {
	Protocol             string `json:"protocol"`
	ExternalPort         int    `json:"external_port"`
	InternalIP           string `json:"internal_ip"`
	InternalPort         int    `json:"internal_port"`
	Description          string `json:"description"`
	LeaseDurationSeconds int    `json:"lease_duration_seconds"`
}

// StatusResponse は、現在のルーター探索状態、WAN接続インターフェース情報、およびアクティブなポートマッピング一覧を網羅した JSON データ構造です。
type StatusResponse struct {
	Discovered  bool                  `json:"discovered"`
	ServiceType string                `json:"service_type,omitempty"`
	ControlURL  string                `json:"control_url,omitempty"`
	ExternalIP  string                `json:"external_ip,omitempty"`
	LocalIP     string                `json:"local_ip,omitempty"`
	Ports       []PortMappingResponse `json:"ports"`
}

func (r PortMappingRequest) toPortMapping() application.PortMapping {
	return application.PortMapping{
		Protocol:             r.Protocol,
		ExternalPort:         r.ExternalPort,
		InternalIP:           r.InternalIP,
		InternalPort:         r.InternalPort,
		Description:          r.Description,
		LeaseDurationSeconds: r.LeaseDurationSeconds,
	}
}

func (r ClosePortRequest) toPortMapping() application.PortMapping {
	return application.PortMapping{
		Protocol:     r.Protocol,
		ExternalPort: r.ExternalPort,
	}
}

func newPortMappingResponse(mapping application.PortMapping) PortMappingResponse {
	return PortMappingResponse{
		Protocol:             mapping.Protocol,
		ExternalPort:         mapping.ExternalPort,
		InternalIP:           mapping.InternalIP,
		InternalPort:         mapping.InternalPort,
		Description:          mapping.Description,
		LeaseDurationSeconds: mapping.LeaseDurationSeconds,
	}
}

func newPortMappingResponses(mappings []application.PortMapping) []PortMappingResponse {
	ports := make([]PortMappingResponse, len(mappings))
	for i, mapping := range mappings {
		ports[i] = newPortMappingResponse(mapping)
	}
	return ports
}

func newStatusResponse(s service.Status) StatusResponse {
	return StatusResponse{
		Discovered:  s.Discovered,
		ServiceType: s.ServiceType,
		ControlURL:  s.ControlURL,
		ExternalIP:  s.ExternalIP,
		LocalIP:     s.LocalIP,
		Ports:       newPortMappingResponses(s.Ports),
	}
}
