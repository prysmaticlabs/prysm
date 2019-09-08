package spectest

// ShuffleTestCase --
type ShuffleTestCase struct {
	Seed    string   `yaml:"seed"`
	Count   uint64   `yaml:"count"`
	Mapping []uint64 `yaml:"mapping"`
}
