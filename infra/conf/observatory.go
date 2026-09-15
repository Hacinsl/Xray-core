package conf

import (
	"strings"

	"github.com/xtls/xray-core/app/observatory"
	"github.com/xtls/xray-core/app/observatory/burst"
	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/infra/conf/cfgcommon/duration"
)

type ObservatoryConfig struct {
	SubjectSelector   []string          `json:"subjectSelector"`
	ProbeURL          string            `json:"probeURL"`
	ProbeInterval     duration.Duration `json:"probeInterval"`
	EnableConcurrency bool              `json:"enableConcurrency"`
}

func (o *ObservatoryConfig) Build() (*observatory.Config, error) {
	return &observatory.Config{
		SubjectSelector:   o.SubjectSelector,
		ProbeUrl:          o.ProbeURL,
		ProbeInterval:     int64(o.ProbeInterval),
		EnableConcurrency: o.EnableConcurrency,
	}, nil
}

// healthCheckSettings holds settings for health Checker
type HealthCheckSettings struct {
	Destination   string            `json:"destination"`
	Connectivity  string            `json:"connectivity"`
	Interval      duration.Duration `json:"interval"`
	SamplingCount int               `json:"sampling"`
	Timeout       duration.Duration `json:"timeout"`
	HttpMethod    string            `json:"httpMethod"`
}

func (h HealthCheckSettings) Build() (*burst.HealthPingConfig, error) {
	var httpMethod string
	if h.HttpMethod == "" {
		httpMethod = "HEAD"
	} else {
		httpMethod = strings.TrimSpace(h.HttpMethod)
	}
	return &burst.HealthPingConfig{
		Destination:   h.Destination,
		Connectivity:  h.Connectivity,
		Interval:      int64(h.Interval),
		Timeout:       int64(h.Timeout),
		SamplingCount: int32(h.SamplingCount),
		HttpMethod:    httpMethod,
	}, nil
}

type BurstObservatoryConfig struct {
	SubjectSelector []string `json:"subjectSelector"`
	// health check settings
	HealthCheck *HealthCheckSettings `json:"pingConfig,omitempty"`
}

func (b BurstObservatoryConfig) Build() (*burst.Config, error) {
	if b.HealthCheck == nil {
		return nil, errors.New("BurstObservatory requires a valid pingConfig")
	}
	if result, err := b.HealthCheck.Build(); err == nil {
		return &burst.Config{
			SubjectSelector: b.SubjectSelector,
			PingConfig:      result,
		}, nil
	} else {
		return nil, err
	}
}
