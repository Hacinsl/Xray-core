package conf

import (
	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/core"
)

type ConfigureFilePreProcessingStage func(conf *Config) error

var configureFilePreProcessingStages []ConfigureFilePreProcessingStage

func RegisterConfigureFilePreProcessingStage(stage ConfigureFilePreProcessingStage) {
	configureFilePreProcessingStages = append(configureFilePreProcessingStages, stage)
}

func PreProcessConfigureFile(conf *Config) error {
	for _, stage := range configureFilePreProcessingStages {
		if err := stage(conf); err != nil {
			return errors.New("Rejected by Preprocessing Stage").Base(err)
		}
	}
	return nil
}

type ConfigureFilePostProcessingStage func(conf *core.Config) error

var configureFilePostProcessingStages []ConfigureFilePostProcessingStage

func RegisterConfigureFilePostProcessingStage(stage ConfigureFilePostProcessingStage) {
	configureFilePostProcessingStages = append(configureFilePostProcessingStages, stage)
}

func PostProcessConfigureFile(conf *core.Config) error {
	for _, stage := range configureFilePostProcessingStages {
		if err := stage(conf); err != nil {
			return errors.New("Rejected by Postprocessing Stage").Base(err)
		}
	}
	return nil
}
