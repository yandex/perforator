package bundle

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"

	tasks "github.com/yandex/perforator/perforator/internal/asynctask/compound"
	binary "github.com/yandex/perforator/perforator/pkg/storage/binary"
	"github.com/yandex/perforator/perforator/pkg/storage/databases"
	"github.com/yandex/perforator/perforator/pkg/storage/microscope"
	profile "github.com/yandex/perforator/perforator/pkg/storage/profile"
)

type Config struct {
	ProfileStorage    *profile.Config                   `yaml:"profiles"`
	BinaryStorage     *binary.Config                    `yaml:"binaries"`
	MicroscopeStorage *microscope.MicroscopeStorageType `yaml:"microscope"`
	TaskStorage       *tasks.TasksConfig                `yaml:"tasks"`

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
