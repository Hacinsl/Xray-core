package conf

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/transport/internet"
	"github.com/xtls/xray-core/transport/internet/hysteria/congestion/bbr"
)

type TransportMethod string

var methodSettingsLoader = NewJSONConfigLoader(ConfigCreatorCache{
	"tcp":         func() interface{} { return new(TCPConfig) },
	"splithttp":   func() interface{} { return new(SplitHTTPConfig) },
	"mkcp":        func() interface{} { return new(KCPConfig) },
	"grpc":        func() interface{} { return new(GRPCConfig) },
	"websocket":   func() interface{} { return new(WebSocketConfig) },
	"httpupgrade": func() interface{} { return new(HttpUpgradeConfig) },
	"hysteria":    func() interface{} { return new(HysteriaConfig) },
	"xdrive":      func() interface{} { return new(XDriveConfig) },
}, "method", "settings")

var securitySettingsLoader = NewJSONConfigLoader(ConfigCreatorCache{
	"tls":     func() interface{} { return new(TLSConfig) },
	"reality": func() interface{} { return new(REALITYConfig) },
}, "security", "settings")

// Build implements Buildable.
func (p TransportMethod) Build() (string, error) {
	switch strings.ToLower(string(p)) {
	case "raw", "tcp":
		return "tcp", nil
	case "xhttp", "splithttp":
		return "splithttp", nil
	case "kcp", "mkcp":
		return "mkcp", nil
	case "grpc":
		errors.PrintNonRemovalDeprecatedFeatureWarning("gRPC transport (with unnecessary costs, etc.)", "XHTTP stream-up H2")
		return "grpc", nil
	case "ws", "websocket":
		errors.PrintNonRemovalDeprecatedFeatureWarning("WebSocket transport (with ALPN http/1.1, etc.)", "XHTTP H2 & H3")
		return "websocket", nil
	case "httpupgrade":
		errors.PrintNonRemovalDeprecatedFeatureWarning("HTTPUpgrade transport (with ALPN http/1.1, etc.)", "XHTTP H2 & H3")
		return "httpupgrade", nil
	case "h2", "h3", "http":
		return "", errors.PrintRemovedFeatureError("HTTP transport (without header padding, etc.)", "XHTTP stream-one H2 & H3")
	case "quic":
		return "", errors.PrintRemovedFeatureError("QUIC transport (without web service, etc.)", "XHTTP stream-one H3")
	case "hysteria":
		return "hysteria", nil
	case "xdrive":
		return "xdrive", nil
	default:
		return "", errors.New("Config: unknown transport method: ", p)
	}
}

// 多态 method + methodSettings，以及 security + securitySettings
type StreamConfig struct {
	Address          *Address         `json:"address"`
	Port             uint16           `json:"port"`
	Method           *TransportMethod `json:"method"`
	Security         string           `json:"security"`
	MethodSettings   *json.RawMessage `json:"methodSettings"`
	SecuritySettings *json.RawMessage `json:"securitySettings"`
	FinalMask        *FinalMask       `json:"finalmask"`
	SocketSettings   *SocketConfig    `json:"sockopt"`
}

// Build implements Buildable.
func (c *StreamConfig) Build() (*internet.StreamConfig, error) {
	config := &internet.StreamConfig{
		Port:       uint32(c.Port),
		MethodName: "tcp",
	}
	if c.Address != nil {
		config.Address = c.Address.Build()
	}
	if c.Method != nil {
		method, err := c.Method.Build()
		if err != nil {
			return nil, err
		}
		config.MethodName = method

		if c.Security == "reality" && method != "tcp" && method != "splithttp" && method != "grpc" {
			return nil, errors.New("REALITY only supports RAW, XHTTP and gRPC for now.")
		}

		methodSettings := []byte("{}")
		if c.MethodSettings != nil {
			methodSettings = ([]byte)(*c.MethodSettings)
		}
		rawConfig, err := methodSettingsLoader.LoadWithID(methodSettings, method)
		if err != nil {
			return nil, errors.New("Failed to load method config for ", method).Base(err)
		}
		ts, err := rawConfig.(Buildable).Build()
		if err != nil {
			return nil, errors.New("Failed to build method config for ", method).Base(err)
		}
		config.TransportSettings = append(config.TransportSettings, &internet.TransportConfig{
			MethodName: method,
			Settings:   serial.ToTypedMessage(ts),
		})
	}

	if c.Security != "" && c.Security != "none" {
		securitySettings := []byte("{}")
		if c.SecuritySettings != nil {
			securitySettings = ([]byte)(*c.SecuritySettings)
		}
		rawConfig, err := securitySettingsLoader.LoadWithID(securitySettings, c.Security)
		if err != nil {
			return nil, errors.New("Failed to load security config for ", c.Security).Base(err)
		}
		ts, err := rawConfig.(Buildable).Build()
		if err != nil {
			return nil, errors.New("Failed to build method config for ", c.Security).Base(err)
		}
		tm := serial.ToTypedMessage(ts)
		config.SecuritySettings = append(config.SecuritySettings, tm)
		config.SecurityType = tm.Type
	}
	if c.SocketSettings != nil {
		ss, err := c.SocketSettings.Build()
		if err != nil {
			return nil, errors.New("Failed to build sockopt.").Base(err)
		}
		config.SocketSettings = ss
	}

	if c.FinalMask != nil {
		for _, mask := range c.FinalMask.Tcp {
			u, err := mask.Build(true)
			if err != nil {
				return nil, errors.New("failed to build mask with type ", mask.Type).Base(err)
			}
			config.Tcpmasks = append(config.Tcpmasks, serial.ToTypedMessage(u))
		}
		for _, mask := range c.FinalMask.Udp {
			u, err := mask.Build(false)
			if err != nil {
				return nil, errors.New("failed to build mask with type ", mask.Type).Base(err)
			}
			config.Udpmasks = append(config.Udpmasks, serial.ToTypedMessage(u))
		}
		if c.FinalMask.QuicParams != nil {
			profile := strings.ToLower(c.FinalMask.QuicParams.BbrProfile)
			switch profile {
			case "", string(bbr.ProfileConservative), string(bbr.ProfileStandard), string(bbr.ProfileAggressive):
				if profile == "" {
					profile = string(bbr.ProfileStandard)
				}
			default:
				return nil, errors.New("unknown bbr profile")
			}

			up, err := c.FinalMask.QuicParams.BrutalUp.Bps()
			if err != nil {
				return nil, err
			}
			down, err := c.FinalMask.QuicParams.BrutalDown.Bps()
			if err != nil {
				return nil, err
			}

			if up > 0 && up < 65536 {
				return nil, errors.New("BrutalUp must be at least 65536 bytes per second")
			}
			if down > 0 && down < 65536 {
				return nil, errors.New("BrutalDown must be at least 65536 bytes per second")
			}

			c.FinalMask.QuicParams.Congestion = strings.ToLower(c.FinalMask.QuicParams.Congestion)
			switch c.FinalMask.QuicParams.Congestion {
			case "", "brutal", "reno", "bbr":
			case "force-brutal":
				if up == 0 {
					return nil, errors.New("force-brutal requires up")
				}
			default:
				return nil, errors.New("unknown congestion control: ", c.FinalMask.QuicParams.Congestion, ", valid values: reno, bbr, brutal, force-brutal")
			}

			if c.FinalMask.QuicParams.InitStreamReceiveWindow > 0 && c.FinalMask.QuicParams.InitStreamReceiveWindow < 16384 {
				return nil, errors.New("InitStreamReceiveWindow must be at least 16384")
			}
			if c.FinalMask.QuicParams.MaxStreamReceiveWindow > 0 && c.FinalMask.QuicParams.MaxStreamReceiveWindow < 16384 {
				return nil, errors.New("MaxStreamReceiveWindow must be at least 16384")
			}
			if c.FinalMask.QuicParams.InitConnectionReceiveWindow > 0 && c.FinalMask.QuicParams.InitConnectionReceiveWindow < 16384 {
				return nil, errors.New("InitConnectionReceiveWindow must be at least 16384")
			}
			if c.FinalMask.QuicParams.MaxConnectionReceiveWindow > 0 && c.FinalMask.QuicParams.MaxConnectionReceiveWindow < 16384 {
				return nil, errors.New("MaxConnectionReceiveWindow must be at least 16384")
			}
			if c.FinalMask.QuicParams.MaxIdleTimeout != 0 && (c.FinalMask.QuicParams.MaxIdleTimeout < 4 || c.FinalMask.QuicParams.MaxIdleTimeout > 120) {
				return nil, errors.New("MaxIdleTimeout must be between 4 and 120")
			}
			if c.FinalMask.QuicParams.KeepAlivePeriod != 0 && (c.FinalMask.QuicParams.KeepAlivePeriod < 2 || c.FinalMask.QuicParams.KeepAlivePeriod > 60) {
				return nil, errors.New("KeepAlivePeriod must be between 2 and 60")
			}
			if c.FinalMask.QuicParams.MaxIncomingStreams != 0 && c.FinalMask.QuicParams.MaxIncomingStreams < 8 {
				return nil, errors.New("MaxIncomingStreams must be at least 8")
			}

			if c.FinalMask.QuicParams.Debug {
				os.Setenv("HYSTERIA_BBR_DEBUG", "true")
				os.Setenv("HYSTERIA_BRUTAL_DEBUG", "true")
			}

			config.QuicParams = &internet.QuicParams{
				Congestion:                    c.FinalMask.QuicParams.Congestion,
				BbrProfile:                    profile,
				BrutalUp:                      up,
				BrutalDown:                    down,
				BrutalDisableLossCompensation: c.FinalMask.QuicParams.BrutalDisableLossCompensation,
				InitStreamReceiveWindow:       c.FinalMask.QuicParams.InitStreamReceiveWindow,
				MaxStreamReceiveWindow:        c.FinalMask.QuicParams.MaxStreamReceiveWindow,
				InitConnReceiveWindow:         c.FinalMask.QuicParams.InitConnectionReceiveWindow,
				MaxConnReceiveWindow:          c.FinalMask.QuicParams.MaxConnectionReceiveWindow,
				MaxIdleTimeout:                c.FinalMask.QuicParams.MaxIdleTimeout,
				KeepAlivePeriod:               c.FinalMask.QuicParams.KeepAlivePeriod,
				DisablePathMtuDiscovery:       c.FinalMask.QuicParams.DisablePathMTUDiscovery,
				DisableChromeParrot:           c.FinalMask.QuicParams.DisableChromeParrot,
				DisableGSO:                    c.FinalMask.QuicParams.DisableGSO,
				MaxIncomingStreams:            c.FinalMask.QuicParams.MaxIncomingStreams,
				DisableStatelessReset:         c.FinalMask.QuicParams.DisableStatelessReset,
			}
		}
	}

	return config, nil
}
