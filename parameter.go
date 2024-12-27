package turnip

import (
	"errors"
	"fmt"
	"reflect"

	"go.uber.org/zap"
)

type Parameter interface {
	Name() string
}

type environment struct {
	selectors  []*selector
	candidates []*candidate
	settings   settings
	fallback   *fallback
	logger     *zap.SugaredLogger
}

func newEnv(params []Parameter) (environment, error) {
	env := environment{
		settings: make(settings),
	}

	for _, p := range params {
		if p == nil {
			return environment{}, errors.New("only one default type can be used at a time")
		}

		switch param := p.(type) {
		case *selector:
			env.selectors = append(env.selectors, param)
		case *candidate:
			env.candidates = append(env.candidates, param)
		case setting:
			env.settings[param] = true
		case *fallback:
			if env.fallback != nil {
				return environment{}, errors.New("only one default type can be used at a time")
			}

			env.fallback = param
		default:
			return environment{}, fmt.Errorf("invalid parameter type '%s'", reflect.TypeOf(param).Name())
		}
	}

	if len(env.candidates) == 0 {
		return environment{}, errors.New("at least one candidate must be defined")
	}

	if env.settings.Get(enableVerbose) {
		env.logger = zap.Must(zap.NewDevelopment()).Sugar().Named("turnip")
		return env, nil
	}

	env.logger = zap.NewNop().Sugar()
	return env, nil
}

type settings map[setting]bool

func (s settings) Get(setting setting) bool {
	v, ok := s[setting]
	return ok && v
}

func Candidate(v any) Parameter {
	return &candidate{
		typ: reflect.TypeOf(v),
	}
}

type candidate struct {
	typ reflect.Type
}

func (c *candidate) Name() string {
	return "Candidate"
}

func SelectOn(field string, equal any, then any) Parameter {
	return &selector{}
}

type selector struct {
}

func (c *selector) Name() string {
	return "Selector"
}

func Default(v any) Parameter {
	return &fallback{}
}

type fallback struct {
}

func (c *fallback) Name() string {
	return "Fallback"
}

func EnableDebug() Parameter {
	return enableVerbose
}

type setting uint

const (
	enableVerbose setting = iota
)

func (s setting) Name() string {
	return "Setting"
}
