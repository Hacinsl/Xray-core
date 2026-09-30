package masque

import (
	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/transport/internet"
)

const protocolName = "masque"

const DefaultPath = "/.well-known/masque/ip/*/*/"

func init() {
	common.Must(internet.RegisterMethodConfigCreator(protocolName, func() interface{} {
		return &Config{
			Path: DefaultPath,
		}
	}))
}
