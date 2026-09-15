package wireguard

import (
	"context"

	"github.com/xtls/xray-core/common"
)

func init() {
	// common.Must(common.RegisterConfig((*DeviceConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
	// 	deviceConfig := config.(*DeviceConfig)
	// 	if deviceConfig.IsClient {
	// 		return NewClient(ctx, deviceConfig)
	// 	} else {
	// 		return NewServer(ctx, deviceConfig)
	// 	}
	// }))

	common.Must(common.RegisterConfig((*InboundConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		inboundConfig := config.(*InboundConfig)
		return NewServer(ctx, inboundConfig)
	}))

	common.Must(common.RegisterConfig((*OutboundConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		outboundConfig := config.(*OutboundConfig)
		return NewClient(ctx, outboundConfig)
	}))
}
