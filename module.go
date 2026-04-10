package ftdc

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.viam.com/rdk/ftdc"
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
	FTDCDirectory *string `json:"ftdc_directory"`
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

type ftdcService struct {
	resource.AlwaysRebuild

	name    resource.Name
	ftdcDir string

	cancelCtx  context.Context
	cancelFunc func()
	logger     logging.Logger
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
	service := &ftdcService{
		name:       name,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
		logger:     logger,
	}
	if conf.FTDCDirectory != nil {
		service.ftdcDir = *conf.FTDCDirectory
	}

	if service.ftdcDir == "" {
		home, homeOk := os.LookupEnv("VIAM_HOME")
		part, partOk := os.LookupEnv("VIAM_MACHINE_PART_ID")
		if !homeOk || !partOk {
			return nil, fmt.Errorf(
				"FTDC directory unknown. Configuration is empty and the home/part id env variables are unset. HomeSet? %v PartSet? %v",
				homeOk, partOk)
		}
		service.ftdcDir = filepath.Join(home, "diagnostics.data", part)
	}
	logger.Info("FTDCDir:", service.ftdcDir)

	return service, nil
}

func (service *ftdcService) Name() resource.Name {
	return service.name
}

func (service *ftdcService) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	if filters, exists := cmd["get_ftdc"]; exists {
		return service.GetFTDC(filters)
	}

	keys := []string{}
	for key, _ := range cmd {
		keys = append(keys, key)
	}

	return nil, fmt.Errorf("Unknown DoCommand. Given keys: %v Available commands: `get-ftdc`", keys)
}

func (service *ftdcService) getDatums() ([]ftdc.FlatDatum, int64, error) {
	var mostRecentFtdcFile *fs.FileInfo
	err := filepath.Walk(service.ftdcDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		service.logger.Info("Path:", path, "Timestamp:", info.ModTime())
		if !strings.HasSuffix(path, ".ftdc") {
			return nil
		}

		if mostRecentFtdcFile == nil || info.ModTime().Compare((*mostRecentFtdcFile).ModTime()) > 0 {
			mostRecentFtdcFile = &info
		}

		return nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("Error walking ftdc directory. Err: %w", err)
	}

	if mostRecentFtdcFile == nil {
		return nil, 0, fmt.Errorf("No ftdc files found. Directory: %v", service.ftdcDir)
	}

	absoluteFilepath := filepath.Join(service.ftdcDir, (*mostRecentFtdcFile).Name())
	reader, err := os.Open(absoluteFilepath)
	if err != nil {
		return nil, 0, fmt.Errorf("Error opening ftdc file. File: %v Err: %w", absoluteFilepath, err)
	}

	datums, lastTimestampNanos, err := ftdc.ParseWithLogger(reader, service.logger)
	if err != nil {
		// We are reading a file that can be concurrently written to. Errors are expected.
		service.logger.Infof("Error reading file. Num datums: %v LastTimestamp: %v Err:",
			len(datums), lastTimestampNanos, err)
	}

	return datums, lastTimestampNanos, nil
}

func getInt64(inp any, defValue int64, logger logging.Logger) int64 {
	switch n := inp.(type) {
	case nil:
		return defValue
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return int64(n)
	case float32:
		return int64(n)
	case float64:
		return int64(n)
	default:
		logger.Warn("Unexpected numeric input. Type: %T Val: %v", inp, inp)
		return defValue
	}
}

func (service *ftdcService) GetFTDC(filters any) (map[string]interface{}, error) {
	var filtersMap map[string]any
	var ok bool
	if filtersMap, ok = filters.(map[string]any); !ok {
		return nil, fmt.Errorf("Filters is not a json map. Type: %T", filters)
	}

	allDatums, lastTimestampNanos, err := service.getDatums()
	if err != nil {
		return nil, err
	}
	_ = lastTimestampNanos

	recentTimeSecs := getInt64(filtersMap["recent_time_secs"], int64(600), service.logger)
	ret := make([]ftdc.FlatDatum, 0, len(allDatums))
	for _, datum := range allDatums {
		if datum.Time >= time.Now().Add(-time.Duration(recentTimeSecs)*time.Second).UnixNano() {
			ret = append(ret, datum)
		}
	}

	return map[string]any{
		"datums": ret,
	}, nil
}

func (service *ftdcService) Status(ctx context.Context) (map[string]interface{}, error) {
	return nil, fmt.Errorf("not implemented")
}

func (service *ftdcService) Close(context.Context) error {
	// Put close code here
	service.cancelFunc()
	return nil
}
