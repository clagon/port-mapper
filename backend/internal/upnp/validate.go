package upnp

import (
	"fmt"
	"net"
	"strings"
)

// ValidatePortMapping は、入力されたポートマッピング要求のプロトコル、ポート番号、IPアドレス、リースタイムが、UPnPのプロトコル規則に合致しているかを事前にチェックします。
func ValidatePortMapping(m PortMapping) error {
	if err := validateProtocol(m.Protocol); err != nil {
		return err
	}
	if err := validatePort("external port", m.ExternalPort); err != nil {
		return err
	}
	if err := validatePort("internal port", m.InternalPort); err != nil {
		return err
	}
	if err := validateInternalIP(m.InternalIP); err != nil {
		return err
	}
	if err := validateLeaseDuration(m.LeaseDurationSeconds); err != nil {
		return err
	}
	return nil
}

func validateProtocol(protocol string) error {
	switch strings.ToUpper(strings.TrimSpace(protocol)) {
	case "TCP", "UDP":
		return nil
	default:
		return fmt.Errorf("invalid protocol %q: must be TCP or UDP", protocol)
	}
}

func validatePort(label string, port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%s %d out of range: must be 1-65535", label, port)
	}
	return nil
}

func validateInternalIP(ip string) error {
	if strings.TrimSpace(ip) == "" {
		return fmt.Errorf("internal ip is required")
	}
	if parsed := net.ParseIP(ip); parsed == nil {
		return fmt.Errorf("invalid internal ip %q", ip)
	}
	return nil
}

func validateLeaseDuration(seconds int) error {
	if seconds < 0 {
		return fmt.Errorf("lease duration must be >= 0")
	}
	if seconds > MaxLeaseDurationSeconds {
		return fmt.Errorf("lease duration %d exceeds max %d", seconds, MaxLeaseDurationSeconds)
	}
	return nil
}
