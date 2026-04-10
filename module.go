package ftdc

import (
	"context"
	"errors"
	"fmt"

	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	generic "go.viam.com/rdk/services/generic"
)

var (
	Ftdc             = resource.NewModel("dan", "ftdc", "ftdc")
	errUnimplemented = errors.New("unimplemented")
)

func init() {
	resource.RegisterService(generic.API, Ftdc,
		resource.Registration[resource.Resource, *Config]{
			Constructor: newFtdcFtdc,
		},
	)
}

type Config struct {
	/*
		Put config attributes here. There should be public/exported fields
		with a `json` parameter at the end of each attribute.

		Example config struct:
			type Config struct {
				Pin   string `json:"pin"`
				Board string `json:"board"`
				MinDeg *float64 `json:"min_angle_deg,omitempty"`
			}

		If your model does not need a config, replace *Config in the init
		function with resource.NoNativeConfig
	*/
}

// Validate ensures all parts of the config are valid and important fields exist.
// Returns three values:
//  1. Required dependencies: other resources that must exist for this resource to work.
//  2. Optional dependencies: other resources that may exist but are not required.
//  3. An error if any Config fields are missing or invalid.
//
// The `path` parameter indicates
// where this resource appears in the machine's JSON configuration
// (for example, "components.0"). You can use it in error messages
// to indicate which resource has a problem.
func (cfg *Config) Validate(path string) ([]string, []string, error) {
	// Add config validation code here
	return nil, nil, nil
}

type ftdcFtdc struct {
	resource.AlwaysRebuild

	name resource.Name

	logger logging.Logger
	cfg    *Config

	cancelCtx  context.Context
	cancelFunc func()
}

func newFtdcFtdc(ctx context.Context, deps resource.Dependencies, rawConf resource.Config, logger logging.Logger) (resource.Resource, error) {
	conf, err := resource.NativeConfig[*Config](rawConf)
	if err != nil {
		return nil, err
	}

	return NewFtdc(ctx, deps, rawConf.ResourceName(), conf, logger)

}

func NewFtdc(ctx context.Context, deps resource.Dependencies, name resource.Name, conf *Config, logger logging.Logger) (resource.Resource, error) {

	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	s := &ftdcFtdc{
		name:       name,
		logger:     logger,
		cfg:        conf,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
	}
	return s, nil
}

func (s *ftdcFtdc) Name() resource.Name {
	return s.name
}

func (s *ftdcFtdc) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *ftdcFtdc) Status(ctx context.Context) (map[string]interface{}, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *ftdcFtdc) Close(context.Context) error {
	// Put close code here
	s.cancelFunc()
	return nil
}
