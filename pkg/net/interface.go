package net

import (
	"errors"
	"net"
	"regexp"
)

func LAN() (string, error) {
	m, err := Interfaces(false)
	if err != nil {
		return "", err
	}

	for _, v := range m {
		return v, nil
	}

	return "localhost", err
}

// Interfaces returns a `name:ip` map of the suitable interfaces found
func Interfaces(listAll bool) (map[string]string, error) {
	names := make(map[string]string)
	ifaces, err := net.Interfaces()
	if err != nil {
		return names, err
	}
	re := regexp.MustCompile(`^(veth|br\-|docker|lo|EHC|XHC|bridge|gif|stf|p2p|awdl|utun|tun|tap)`)
	for _, iface := range ifaces {
		if !listAll && re.MatchString(iface.Name) {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		ip, err := FindIP(iface)
		if err != nil {
			continue
		}
		names[iface.Name] = ip
	}
	return names, nil
}

// FindIP returns the IP address of the passed interface, and an error
func FindIP(iface net.Interface) (string, error) {
	var ip string
	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ipnet.IP.IsLinkLocalUnicast() {
				continue
			}
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
				continue
			}
			// Use IPv6 only if an IPv4 hasn't been found yet.
			// This is eventually overwritten with an IPv4, if found (see above)
			if ip == "" {
				ip = "[" + ipnet.IP.String() + "]"
			}
		}
	}
	if ip == "" {
		return "", errors.New("unable to find an IP for this interface")
	}
	return ip, nil
}
