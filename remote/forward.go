package remote

import (
	"net"
	"net/url"
	"strconv"
	"strings"
)

func forward(creds map[string]string, toPort uint) (fromAddress string) {
	if creds == nil {
		return ""
	}

	forwardedPort := strconv.FormatUint(uint64(toPort), 10)
	override := map[string]string{}
	host, port := creds["hostname"], creds["port"]
	if host != "" || port != "" {
		override["port"] = forwardedPort
	}
	if host != "" {
		override["hostname"] = "localhost"
	}

	if creds["uri"] != "" {
		u, err := url.Parse(creds["uri"])
		if err != nil || u.Host == "" {
			return ""
		}
		host, port = ensureHostPort(host, port, u.Host)
		u.Host = "localhost:" + forwardedPort
		override["uri"] = u.String()
	}

	if creds["jdbcUrl"] != "" {
		u, err := url.Parse(strings.TrimPrefix(creds["jdbcUrl"], "jdbc:"))
		if err != nil || u.Host == "" {
			return ""
		}
		host, port = ensureHostPort(host, port, u.Host)
		u.Host = "localhost:" + forwardedPort
		override["jdbcUrl"] = "jdbc:" + u.String()
	}

	if host == "" || port == "" {
		return ""
	}
	merge(override, creds)
	return host + ":" + port
}

func ensureHostPort(knownHost, knownPort, address string) (host, port string) {
	if h, p, err := net.SplitHostPort(address); err == nil {
		host, port = h, p
	}
	if knownHost != "" {
		host = knownHost
	}
	if knownPort != "" {
		port = knownPort
	}
	return
}

func merge(from, to map[string]string) {
	for k, v := range from {
		to[k] = v
	}
}
