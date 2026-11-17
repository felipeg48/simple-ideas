package serve

import "net"

// LANIPs are this machine's IPv4 addresses a phone could reach, in interface
// order (Wi-Fi or Ethernet first on a Mac): private ranges, then the shared
// 100.64/10 range Tailscale and carrier networks use.
func LANIPs() []string {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil
	}
	cgnat := &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}
	var private, shared []string
	for _, i := range ifs {
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := i.Addrs()
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			if ip := ipn.IP.To4(); ip != nil && ip.IsPrivate() {
				private = append(private, ip.String())
			} else if ip != nil && cgnat.Contains(ip) {
				shared = append(shared, ip.String())
			}
		}
	}
	return append(private, shared...)
}
