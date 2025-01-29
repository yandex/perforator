package binary

type MetaStorageType string

const (
	PostgresMetaStorage MetaStorageType = "postgres"
)

type Config struct {
	MetaStorage  MetaStorageType `yaml:"meta"`
	S3Bucket     string          `yaml:"bucket"`
	GSYMS3Bucket string          `yaml:"gsym_bucket"`
}
