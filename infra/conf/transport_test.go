package conf_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	. "github.com/xtls/xray-core/infra/conf"
	"github.com/xtls/xray-core/transport/internet"
	finalmaskcustom "github.com/xtls/xray-core/transport/internet/finalmask/header/custom"
	"google.golang.org/protobuf/proto"
)

func TestSocketConfig(t *testing.T) {
	createParser := func() func(string) (proto.Message, error) {
		return func(s string) (proto.Message, error) {
			config := new(SocketConfig)
			if err := json.Unmarshal([]byte(s), config); err != nil {
				return nil, err
			}
			return config.Build()
		}
	}

	// test "tcpFastOpen": true, queue length 256 is expected. other parameters are tested here too
	expectedOutput := &internet.SocketConfig{
		Mark:           1,
		Tfo:            256,
		DomainStrategy: internet.DomainStrategy_USE_IP,
		DialerProxy:    "tag",
		HappyEyeballs:  &internet.HappyEyeballsConfig{Interleave: 1, TryDelayMs: 0, PrioritizeIpv6: false, MaxConcurrentTry: 4},
	}
	runMultiTestCase(t, []TestCase{
		{
			Input: `{
				"mark": 1,
				"tcpFastOpen": true,
				"domainStrategy": "UseIP",
				"dialerProxy": "tag"
			}`,
			Parser: createParser(),
			Output: expectedOutput,
		},
	})
	if expectedOutput.ParseTFOValue() != 256 {
		t.Fatalf("unexpected parsed TFO value, which should be 256")
	}

	// test "tcpFastOpen": false, disabled TFO is expected
	expectedOutput = &internet.SocketConfig{
		Mark:          0,
		Tfo:           -1,
		HappyEyeballs: &internet.HappyEyeballsConfig{Interleave: 1, TryDelayMs: 0, PrioritizeIpv6: false, MaxConcurrentTry: 4},
	}
	runMultiTestCase(t, []TestCase{
		{
			Input: `{
				"tcpFastOpen": false
			}`,
			Parser: createParser(),
			Output: expectedOutput,
		},
	})
	if expectedOutput.ParseTFOValue() != 0 {
		t.Fatalf("unexpected parsed TFO value, which should be 0")
	}

	// test "tcpFastOpen": 65535, queue length 65535 is expected
	expectedOutput = &internet.SocketConfig{
		Mark:          0,
		Tfo:           65535,
		HappyEyeballs: &internet.HappyEyeballsConfig{Interleave: 1, TryDelayMs: 0, PrioritizeIpv6: false, MaxConcurrentTry: 4},
	}
	runMultiTestCase(t, []TestCase{
		{
			Input: `{
				"tcpFastOpen": 65535
			}`,
			Parser: createParser(),
			Output: expectedOutput,
		},
	})
	if expectedOutput.ParseTFOValue() != 65535 {
		t.Fatalf("unexpected parsed TFO value, which should be 65535")
	}

	// test "tcpFastOpen": -65535, disable TFO is expected
	expectedOutput = &internet.SocketConfig{
		Mark:          0,
		Tfo:           -65535,
		HappyEyeballs: &internet.HappyEyeballsConfig{Interleave: 1, TryDelayMs: 0, PrioritizeIpv6: false, MaxConcurrentTry: 4},
	}
	runMultiTestCase(t, []TestCase{
		{
			Input: `{
				"tcpFastOpen": -65535
			}`,
			Parser: createParser(),
			Output: expectedOutput,
		},
	})
	if expectedOutput.ParseTFOValue() != 0 {
		t.Fatalf("unexpected parsed TFO value, which should be 0")
	}

	// test "tcpFastOpen": 0, no operation is expected
	expectedOutput = &internet.SocketConfig{
		Mark:          0,
		Tfo:           0,
		HappyEyeballs: &internet.HappyEyeballsConfig{Interleave: 1, TryDelayMs: 0, PrioritizeIpv6: false, MaxConcurrentTry: 4},
	}
	runMultiTestCase(t, []TestCase{
		{
			Input: `{
				"tcpFastOpen": 0
			}`,
			Parser: createParser(),
			Output: expectedOutput,
		},
	})
	if expectedOutput.ParseTFOValue() != -1 {
		t.Fatalf("unexpected parsed TFO value, which should be -1")
	}

	// test omit "tcpFastOpen", no operation is expected
	expectedOutput = &internet.SocketConfig{
		Mark:          0,
		Tfo:           0,
		HappyEyeballs: &internet.HappyEyeballsConfig{Interleave: 1, TryDelayMs: 0, PrioritizeIpv6: false, MaxConcurrentTry: 4},
	}
	runMultiTestCase(t, []TestCase{
		{
			Input:  `{}`,
			Parser: createParser(),
			Output: expectedOutput,
		},
	})
	if expectedOutput.ParseTFOValue() != -1 {
		t.Fatalf("unexpected parsed TFO value, which should be -1")
	}

	// test "tcpFastOpen": null, no operation is expected
	expectedOutput = &internet.SocketConfig{
		Mark:          0,
		Tfo:           0,
		HappyEyeballs: &internet.HappyEyeballsConfig{Interleave: 1, TryDelayMs: 0, PrioritizeIpv6: false, MaxConcurrentTry: 4},
	}
	runMultiTestCase(t, []TestCase{
		{
			Input: `{
				"tcpFastOpen": null
			}`,
			Parser: createParser(),
			Output: expectedOutput,
		},
	})
	if expectedOutput.ParseTFOValue() != -1 {
		t.Fatalf("unexpected parsed TFO value, which should be -1")
	}
}

func TestHeaderCustomUDPBuild(t *testing.T) {
	parser := loadJSON(func() Buildable { return new(HeaderCustomUDP) })

	runMultiTestCase(t, []TestCase{
		{
			Input: `{
				"client": [
					{
						"type": "hex",
						"packet": "aabb"
					},
					{
						"rand": 2,
						"capture": "seed",
						"randRange": "16-32"
					}
				],
				"server": [
					{
						"capture": "txid",
						"transform": {
							"op": "concat",
							"args": [
								{"reuse": "seed"},
								{"u64": 258},
								{"type": "hex", "bytes": "c0de"}
							]
						}
					},
					{
						"reuse": "txid"
					}
				],
				"mode": "standalone"
			}`,
			Parser: parser,
			Output: &finalmaskcustom.UDPStandaloneConfig{
				Client: []*finalmaskcustom.UDPItem{
					{
						RandMax: 255,
						Packet:  []byte{0xAA, 0xBB},
					},
					{
						Rand:    2,
						RandMin: 16,
						RandMax: 32,
						Save:    "seed",
					},
				},
				Server: []*finalmaskcustom.UDPItem{
					{
						RandMax: 255,
						Save:    "txid",
						Expr: &finalmaskcustom.Expr{
							Op: "concat",
							Args: []*finalmaskcustom.ExprArg{
								{
									Value: &finalmaskcustom.ExprArg_Var{
										Var: "seed",
									},
								},
								{
									Value: &finalmaskcustom.ExprArg_U64{
										U64: 258,
									},
								},
								{
									Value: &finalmaskcustom.ExprArg_Bytes{
										Bytes: []byte{0xC0, 0xDE},
									},
								},
							},
						},
					},
					{
						RandMax: 255,
						Var:     "txid",
					},
				},
			},
		},
	})
}

func TestHeaderCustomTCPBuildRejectsMixedItemKinds(t *testing.T) {
	parser := loadJSON(func() Buildable { return new(HeaderCustomTCP) })

	_, err := parser(`{
		"clients": [[
			{
				"packet": [1, 2],
				"reuse": "txid"
			}
		]]
	}`)
	if err == nil || !strings.Contains(err.Error(), "exactly one item kind") {
		t.Fatalf("expected mixed item kind rejection, got %v", err)
	}
}

func TestHeaderCustomUDPBuildRejectsInvalidVariableNames(t *testing.T) {
	parser := loadJSON(func() Buildable { return new(HeaderCustomUDP) })

	_, err := parser(`{
		"client": [
			{
				"capture": "bad-name",
				"rand": 4
			}
		]
	}`)
	if err == nil || !strings.Contains(err.Error(), "invalid variable name") {
		t.Fatalf("expected invalid variable name rejection, got %v", err)
	}
}

func TestHeaderCustomUDPBuildRejectsExprWithoutArgs(t *testing.T) {
	parser := loadJSON(func() Buildable { return new(HeaderCustomUDP) })

	_, err := parser(`{
		"client": [
			{
				"transform": {
					"op": "concat"
				}
			}
		]
	}`)
	if err == nil || !strings.Contains(err.Error(), "transform args") {
		t.Fatalf("expected transform arg rejection, got %v", err)
	}
}

func TestXDriveStreamConfig(t *testing.T) {
	config := new(StreamConfig)
	if err := json.Unmarshal([]byte(`{
		"method": "xdrive",
		"methodSettings": {
			"remoteFolder": "/tmp/xdrive",
			"service": "local"
		}
	}`), config); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	built, err := config.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if built.MethodName != "xdrive" {
		t.Fatalf("MethodName is %q, want %q", built.MethodName, "xdrive")
	}
	if len(built.TransportSettings) != 1 || built.TransportSettings[0].MethodName != "xdrive" {
		t.Fatalf("TransportSettings is %v, want a single xdrive entry", built.TransportSettings)
	}
}

func TestXDriveRejectsUnknownService(t *testing.T) {
	config := new(XDriveConfig)
	if err := json.Unmarshal([]byte(`{"remoteFolder": "/tmp/xdrive", "service": "Dropbox"}`), config); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, err := config.Build(); err == nil {
		t.Fatal("Build accepted an unsupported service")
	}
}

func TestXDriveTemplateStreamConfig(t *testing.T) {
	config := new(StreamConfig)
	if err := json.Unmarshal([]byte(`{
		"method": "xdrive",
		"methodSettings": {
			"remoteFolder": "folder",
			"service": "template",
			"secrets": ["user", "pass"],
			"template": {
				"flatten": true,
				"auth": {"type": "basic", "username": "{secret0}", "password": "{secret1}"},
				"put": {"method": "PUT", "url": "https://dav.example/{folder}/{name}"},
				"get": {"method": "GET", "url": "https://dav.example/{folder}/{name}"},
				"delete": {"method": "DELETE", "url": "https://dav.example/{folder}/{name}"},
				"list": {"method": "PROPFIND", "url": "https://dav.example/{folder}/", "namesRegex": "<d:href>/folder/([^<]+)</d:href>"}
			}
		}
	}`), config); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	built, err := config.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if built.MethodName != "xdrive" {
		t.Fatalf("MethodName is %q, want xdrive", built.MethodName)
	}
}

func TestXDriveTemplateNeedsTemplate(t *testing.T) {
	config := new(XDriveConfig)
	if err := json.Unmarshal([]byte(`{"remoteFolder": "f", "service": "template"}`), config); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, err := config.Build(); err == nil {
		t.Fatal("Build accepted a template service without a template")
	}
}

func TestTransportMethodAliases(t *testing.T) {
	cases := []struct {
		method   string
		settings string
		want     string
	}{
		{`"raw"`, `{}`, "tcp"},
		{`"tcp"`, `{}`, "tcp"},
		{`"xhttp"`, `{"path":"/probe"}`, "splithttp"},
		{`"splithttp"`, `{"path":"/probe"}`, "splithttp"},
		{`"kcp"`, `{"mtu":1400}`, "mkcp"},
		{`"mkcp"`, `{"mtu":1400}`, "mkcp"},
		{`"ws"`, `{"path":"/probe"}`, "websocket"},
		{`"WS"`, `{"path":"/probe"}`, "websocket"},
		{`"websocket"`, `{"path":"/probe"}`, "websocket"},
		{`"grpc"`, `{}`, "grpc"},
		{`"httpupgrade"`, `{}`, "httpupgrade"},
		{`"hysteria"`, `{"version":2,"auth":"probe"}`, "hysteria"},
		{`"xdrive"`, `{"remoteFolder":"/tmp/xdrive","service":"local"}`, "xdrive"},
	}
	for _, c := range cases {
		raw := `{"method":` + c.method + `,"methodSettings":` + c.settings + `}`
		config := new(StreamConfig)
		if err := json.Unmarshal([]byte(raw), config); err != nil {
			t.Fatalf("%s: Unmarshal: %v", raw, err)
		}
		built, err := config.Build()
		if err != nil {
			t.Fatalf("%s: Build: %v", raw, err)
		}
		if built.MethodName != c.want {
			t.Fatalf("%s: StreamConfig.MethodName is %q, want %q", raw, built.MethodName, c.want)
		}
		if len(built.TransportSettings) != 1 {
			t.Fatalf("%s: TransportSettings is %v, want a single entry", raw, built.TransportSettings)
		}
		if name := built.TransportSettings[0].GetUnifiedMethodName(); name != c.want {
			t.Fatalf("%s: TransportSettings[0].MethodName is %q, want %q", raw, name, c.want)
		}

		inSettings, err := built.TransportSettings[0].GetTypedSettings()
		if err != nil {
			t.Fatalf("%s: GetTypedSettings: %v", raw, err)
		}
		effective, err := built.GetEffectiveTransportSettings()
		if err != nil {
			t.Fatalf("%s: GetEffectiveTransportSettings: %v", raw, err)
		}
		if !reflect.DeepEqual(inSettings, effective) {
			t.Fatalf("%s: GetEffectiveTransportSettings did not hit TransportSettings (silent fallback)", raw)
		}
		if c.settings != `{}` && reflect.ValueOf(effective).Elem().IsZero() {
			t.Fatalf("%s: transport settings were dropped: %+v", raw, effective)
		}
	}
}
