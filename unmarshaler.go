package turnip

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

var ErrNoMatch = errors.New("no match")

type Unmarshaler struct {
	resolver Resolver
	settings settings
}

func New(params ...Parameter) (*Unmarshaler, error) {
	env, err := newEnv(params)
	if err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	env.logger.Infow("creating new turnip unmarshaler",
		zap.String("settings", fmt.Sprintf("%v", env.settings)))

	resolver, err := newTraverseResolver(env)
	if err != nil {
		return nil, fmt.Errorf("resolver: %w", err)
	}

	return &Unmarshaler{
		resolver: resolver,
	}, nil
}

func (u *Unmarshaler) UnmarshalJSON(b []byte) (any, error) {
	res := gjson.ParseBytes(b)
	if res.Type != gjson.JSON {
		return nil, errors.New("invalid json: not an object")
	}

	typ, err := u.resolver.ResolveJSON(res)
	if err != nil {
		return nil, fmt.Errorf("resolve: %w", err)
	}

	if typ == nil {
		return nil, ErrNoMatch
	}

	v := reflect.New(typ).Interface()
	err = json.Unmarshal(b, v)
	if err != nil {
		return nil, fmt.Errorf("unmarshall: %w", err)
	}

	return v, nil
}
