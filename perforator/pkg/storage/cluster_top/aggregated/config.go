package aggregated

type Config struct {
	AsyncInsert AsyncInsertConfig `yaml:"async_insert"`
	Query       QueryConfig       `yaml:"query"`
}

func (c *Config) FillDefault() {
	c.Query.FillDefault()
}
