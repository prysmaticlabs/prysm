package backend

// ShuffleTest --
type ShuffleTest struct {
	Title     string             `yaml:"title"`
	Summary   string             `yaml:"summary"`
	TestSuite string             `yaml:"test_suite"`
	Fork      string             `yaml:"fork"`
	Version   string             `yaml:"version"`
	TestCases []*ShuffleTestCase `yaml:"test_cases"`
}

// ShuffleTestCase --
type ShuffleTestCase struct {
	Count    uint64   `yaml:"count,flow"`
	Shuffled []uint64 `yaml:"shuffled,flow"`
	Seed     string   `yaml:"seed,flow`
}
