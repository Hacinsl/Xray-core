package udp

import (
	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/transport/internet"
)

func init() {
	common.Must(internet.RegisterMethodConfigCreator(methodName, func() interface{} {
		return new(Config)
	}))
}
