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
	Input  []uint32 `yaml:"input,flow"`
	Output []uint32 `yaml:"output,flow"`
	Seed   string
}
