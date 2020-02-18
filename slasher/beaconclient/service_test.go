package beaconclient

var (
	_ = Notifier(&Service{})
	_ = HistoricalFetcher(&Service{})
	_ = ChainFetcher(&Service{})
)
