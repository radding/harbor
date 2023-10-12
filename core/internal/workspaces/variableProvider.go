package workspaces

import (
	"fmt"
	"os"
	"strconv"

	"github.com/pkg/errors"
	mathparser "github.com/radding/harbor/internal/MathParser"
)

type Provider interface {
	Resolve(variableName string) (*mathparser.Value, error)
}

type variableResolver struct {
	providers map[string]Provider
}

func (e *variableResolver) GetValue(providerName, variableName string) (*mathparser.Value, error) {
	provider, ok := e.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("can't find povider with name %s", providerName)
	}
	val, err := provider.Resolve(variableName)
	if err != nil {
		return nil, errors.Wrapf(err, "error resolving value with provider %s", providerName)
	}
	return val, nil
}

func newVariableResolver() *variableResolver {
	return &variableResolver{
		providers: map[string]Provider{},
	}
}

func (e *variableResolver) registerProvider(name string, provider Provider) {
	e.providers[name] = provider
}

type ProviderFunc func(variableName string) (*mathparser.Value, error)

func (p ProviderFunc) Resolve(variableName string) (*mathparser.Value, error) {
	return p(variableName)
}

func getEnvVariable(variableName string) (*mathparser.Value, error) {
	value, present := os.LookupEnv(variableName)
	if !present {
		return nil, fmt.Errorf("%s not present in environment", variableName)
	}
	floatVal, err := strconv.ParseFloat(value, 64)
	if err == nil {
		return &mathparser.Value{
			Number: &floatVal,
		}, nil
	}
	boolVal, err := strconv.ParseBool(value)
	if err == nil {
		return &mathparser.Value{
			BoolVal: (*mathparser.Boolean)(&boolVal),
		}, nil
	}
	return &mathparser.Value{
		StringValue: &value,
	}, nil
}
