package bundle

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"

	tasks "github.com/yandex/perforator/perforator/internal/asynctask/compound"
	binary "github.com/yandex/perforator/perforator/pkg/storage/binary"
	custom_profiling_operation "github.com/yandex/perforator/perforator/pkg/storage/custom_profiling_operation"
	"github.com/yandex/perforator/perforator/pkg/storage/databases"
	"github.com/yandex/perforator/perforator/pkg/storage/microscope"
	profile "github.com/yandex/perforator/perforator/pkg/storage/profile"
)

type Config struct {
	ProfileStorage                  *profile.Config                                                 `yaml:"profiles"`
	BinaryStorage                   *binary.Config                                                  `yaml:"binaries"`
	MicroscopeStorage               *microscope.MicroscopeStorageType                               `yaml:"microscope"`
	TaskStorage                     *tasks.TasksConfig                                              `yaml:"tasks"`
	CustomProfilingOperationStorage *custom_profiling_operation.CustomProfilingOperationStorageType `yaml:"custom_profiling_operation"`

	DBs databases.Config `yaml:"databases"`
}

func ParseConfig(path string, strict bool) (conf *Config, err error) {
	// TODO(PERFORATOR-480) always be strict
	var file *os.File
	file, err = os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

	conf = &Config{}
	dec := yaml.NewDecoder(file)
	if strict {
		dec.SetStrict(true)
	}
	err = dec.Decode(conf)
	return
}
