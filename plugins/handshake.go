package plugins

import "github.com/hashicorp/go-plugin"

var HandShake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "HarborPlugins",
	MagicCookieValue: "1234567890",
}
