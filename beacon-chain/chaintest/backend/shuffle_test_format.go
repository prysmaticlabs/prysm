package backend

// ShuffleTest --
type ShuffleTest struct {
	Title         string             `yaml:"title"`
	Summary       string             `yaml:"summary"`
	ForksTimeline string             `yaml:"forks_timeline"`
	Forks         []string           `yaml:"forks"`
	Config        string             `yaml:"config"`
	Runner        string             `yaml:"runner"`
	Handler       string             `yaml:"handler"`
	TestCases     []*ShuffleTestCase `yaml:"test_cases"`
}

// ShuffleTestCase --
type ShuffleTestCase struct {
	Seed     string   `yaml:"seed"`
	Count    uint64   `yaml:"count"`
	Shuffled []uint64 `yaml:"shuffled"`
}
