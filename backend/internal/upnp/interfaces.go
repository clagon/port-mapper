package upnp

import "net"

type discoverInterface struct {
	ListenAddr *net.UDPAddr
	IPNet      *net.IPNet
	Interface  *net.Interface
}

type discoverIPv6Interface struct {
	ListenAddr *net.UDPAddr
	Interface  *net.Interface
}

// discoverInterfaces は、ローカルマシン上の有効な IPv4 ネットワークインターフェース情報を列挙します。
// ループバックや停止中のNIC、マルチキャスト非対応NICを除外し、探索ソケットのバインドに適したアドレスリストを生成します。
func discoverInterfaces() ([]discoverInterface, error) {
	var ifaces []discoverInterface
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		ifaceAddrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, ifaceAddr := range ifaceAddrs {
			ip, ipNet := ipv4NetFromAddr(ifaceAddr)
			if ip == nil || ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() {
				continue
			}
			if first := firstUsableIPv4(ipNet); first != nil && first.Equal(ip) {
				continue
			}
			ifaces = append(ifaces, discoverInterface{
				ListenAddr: &net.UDPAddr{IP: ip, Port: 0},
				IPNet:      ipNet,
				Interface:  &iface,
			})
		}
	}
	if len(ifaces) == 0 {
		ifaces = append(ifaces, discoverInterface{
			ListenAddr: &net.UDPAddr{IP: net.IPv4zero, Port: 0},
		})
	}
	return ifaces, nil
}

// discoverIPv6Interfaces は、ローカルマシン上の有効な IPv6 ネットワークインターフェース情報を列挙します。
func discoverIPv6Interfaces() ([]discoverIPv6Interface, error) {
	var ifaces []discoverIPv6Interface
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		ifaceAddrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		hasIPv6 := false
		for _, ifaceAddr := range ifaceAddrs {
			ip := ipv6FromAddr(ifaceAddr)
			if ip == nil || ip.IsLoopback() || ip.IsUnspecified() {
				continue
			}
			hasIPv6 = true
			break
		}
		if hasIPv6 {
			// Windows ではグローバル IPv6 アドレスに zone 付き bind すると失敗するため、NIC 指定は送信側で行う。
			ifaces = append(ifaces, discoverIPv6Interface{
				ListenAddr: &net.UDPAddr{IP: net.IPv6zero, Port: 0},
				Interface:  &iface,
			})
		}
	}
	return ifaces, nil
}

func ipv4NetFromAddr(addr net.Addr) (net.IP, *net.IPNet) {
	switch v := addr.(type) {
	case *net.IPNet:
		return v.IP.To4(), v
	case *net.IPAddr:
		return v.IP.To4(), nil
	default:
		return nil, nil
	}
}

func ipv6FromAddr(addr net.Addr) net.IP {
	var ip net.IP
	switch v := addr.(type) {
	case *net.IPNet:
		ip = v.IP
	case *net.IPAddr:
		ip = v.IP
	default:
		return nil
	}
	if ip.To4() != nil {
		return nil
	}
	return ip
}

func firstUsableIPv4(ipNet *net.IPNet) net.IP {
	if ipNet == nil {
		return nil
	}
	ip := ipNet.IP.To4()
	if ip == nil || len(ipNet.Mask) != net.IPv4len {
		return nil
	}
	network := ip.Mask(ipNet.Mask).To4()
	if network == nil {
		return nil
	}
	return net.IPv4(network[0], network[1], network[2], network[3]+1).To4()
}
