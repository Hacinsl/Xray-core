package conf

import (
	"reflect"

	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/core"
)

type ConfigureFilePreProcessingStage func(conf *Config) error

var configureFilePreProcessingStages = make(map[reflect.Type]ConfigureFilePreProcessingStage)

func RegisterConfigureFilePreProcessingStage(config interface{}, stage ConfigureFilePreProcessingStage) error {
	configType := reflect.TypeOf(config)
	if _, found := configureFilePreProcessingStages[configType]; found {
		return errors.New(configType.String() + " is already registered.")
	}
	configureFilePreProcessingStages[configType] = stage
	return nil
}

func PreProcessConfigureFile(conf *Config) error {
	for configType, stage := range configureFilePreProcessingStages {
		if err := stage(conf); err != nil {
			return errors.New("Rejected by Preprocessing Stage ", configType.String()).Base(err)
		}
	}
	return nil
}

type ConfigureFilePostProcessingStage func(conf *core.Config) error

var configureFilePostProcessingStages = make(map[reflect.Type]ConfigureFilePostProcessingStage)

func RegisterConfigureFilePostProcessingStage(config interface{}, stage ConfigureFilePostProcessingStage) error {
	configType := reflect.TypeOf(config)
	if _, found := configureFilePostProcessingStages[configType]; found {
		return errors.New(configType.String() + " is already registered.")
	}
	configureFilePostProcessingStages[configType] = stage
	return nil
}

func PostProcessConfigureFile(conf *core.Config) error {
	for configType, stage := range configureFilePostProcessingStages {
		if err := stage(conf); err != nil {
			return errors.New("Rejected by Postprocessing Stage ", configType.String()).Base(err)
		}
	}
	return nil
}
