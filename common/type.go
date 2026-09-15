package common

import (
	"context"
	// "fmt"
	"reflect"

	"github.com/xtls/xray-core/common/errors"
)

// ConfigCreator is a function to create an object by a config.
type ConfigCreator func(ctx context.Context, config interface{}) (interface{}, error)

var typeCreatorRegistry = make(map[reflect.Type]ConfigCreator)

// RegisterConfig registers a global config creator. The config can be nil but must have a type.
func RegisterConfig(config interface{}, configCreator ConfigCreator) error {
	configType := reflect.TypeOf(config)
	if _, found := typeCreatorRegistry[configType]; found {
		return errors.New(configType.String() + " is already registered").AtError()
	}
	// fmt.Printf("[note] Register ConfigCreator for %s\n", configType.Elem().PkgPath()+"."+configType.Elem().Name())
	typeCreatorRegistry[configType] = configCreator
	return nil
}

// CreateObject creates an object by its config. The config type must be registered through RegisterConfig().
func CreateObject(ctx context.Context, config interface{}) (interface{}, error) {
	configType := reflect.TypeOf(config)
	creator, found := typeCreatorRegistry[configType]
	if !found {
		return nil, errors.New(configType.String() + " is not registered").AtError()
	}
	// fmt.Printf("[note] Create object for %s\n", configType.Elem().PkgPath()+"."+configType.Elem().Name())
	return creator(ctx, config)
}
