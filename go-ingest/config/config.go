// Package config adds support for loading configuration from multiple yaml files.
package config

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"time"

	"dario.cat/mergo"
	"github.com/go-playground/validator/v10"
	"github.com/mcuadros/go-defaults"
	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	HttpPort           int           `yaml:"HttpPort"`
	DatabaseUrl        string        `yaml:"DatabaseUrl"`
	EmbedderUrl        string        `yaml:"EmbedderUrl"`
	EmbedderReqTimeout time.Duration `yaml:"EmbedderReqTimeout"`
}

func LoadConfig(configFiles []string) *AppConfig {
	var configuration AppConfig
	err := LoadConfiguration(configFiles, &configuration)
	if err != nil {
		log.Panic("error loading configuration:", err)
	}
	return &configuration
}

func LoadConfiguration(configFiles []string, target interface{}) error {
	for _, configFilePath := range configFiles {
		log.Println("Parsing config file", configFilePath)
		rawContent, err := os.ReadFile(configFilePath)
		if err != nil {
			return err
		}
		cfg := newZeroFor(target)
		err = yaml.Unmarshal(rawContent, cfg)
		if err != nil {
			return err
		}
		err = mergo.Merge(target, cfg, mergo.WithOverride)
		if err != nil {
			return err
		}

	}
	return nil
}

// When loading YAML we need a zero value of a specific type in order to drive the parsing, but YAML parser does not
// support deep merging (it will just override at the top level) - so `mergo` is used.
// So this means that we now need a `target` zero value for each of the config files, but we like to keep the public API
// which mimics that of YAML (and JSON parsing). Thus the need for a function that will take a pointer to an arbitrary
// struct type and produce a pointer to a new zero value for that type.
// WARNING: this will crash if passed and interface value to something other than a pointer
func newZeroFor(target interface{}) interface{} {
	return reflect.New(reflect.TypeOf(target).Elem()).Interface()
}

// ValidateConfiguration takes (should take) a struct and validates its fields against predefined `validate` tags.
// The underlying validate.Struct method returns two types of errors. validator.InvalidValidationError for when the
// validation breaks, e.g. when a wrong type is passed as the argument (check validate.StructCtx). In this case, we wrap
// things with a plain error. The other case are actual validation errors. In this case a validator.ValidationErrors is
// returned, meaning our abstraction leaks and the assumption/recommendation is to use only error's Error() method,
// i.e. not to resort to type assertions on the returned instance.
func ValidateConfiguration(target interface{}) error {
	validate := validator.New()
	err := validate.Struct(target)
	if _, ok := err.(*validator.InvalidValidationError); ok {
		return fmt.Errorf("could not validate input (%v): %w", target, err)
	}
	return err
}

// SetDefaultsConfiguration takes a struct and sets its fields to their default values if they are not set.
// The defaults can be set using the `default` tag e.g.
// url string `default:"http://localhost:8080"`
func SetDefaultsConfiguration(target interface{}) {
	defaults.SetDefaults(target)
}

// LoadAndValidateConfiguration is a convenience method that does two logical steps in one go. Make sure to always check
// for errors returned, certain fields might be loaded while others could fail.
func LoadAndValidateConfiguration(configFiles []string, target interface{}) (err error) {
	err = LoadConfiguration(configFiles, target)
	if err != nil {
		return
	}
	err = ValidateConfiguration(target)
	return
}

// LoadSetDefaultsAndValidateConfiguration is a convenience method that does three logical steps in one go. Make sure to always check
// for errors returned, certain fields might be loaded while others could fail.
func LoadSetDefaultsAndValidateConfiguration(configFiles []string, target interface{}) (err error) {
	err = LoadConfiguration(configFiles, target)
	if err != nil {
		return
	}
	SetDefaultsConfiguration(target)
	err = ValidateConfiguration(target)
	return
}
