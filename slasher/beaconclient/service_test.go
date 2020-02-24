package beaconclient

var (
	_ = Notifier(&Service{})
	_ = ChainFetcher(&Service{})
)
