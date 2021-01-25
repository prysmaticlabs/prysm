package slasher

// Config parameters for slashing detection.
type Config struct {
	ChunkSize          uint64
	ValidatorChunkSize uint64
	HistoryLength      uint64
}

// DefaultConfig --
func DefaultConfig() *Config {
	return &Config{
		ChunkSize:          16,
		ValidatorChunkSize: 256,
		HistoryLength:      4096,
	}
}

func (c *Config) chunkIndex(epoch uint64) uint64 {
	return (epoch % c.HistoryLength) / c.ChunkSize
}

func (c *Config) validatorChunkIndex(validatorIndex uint64) uint64 {
	return validatorIndex / c.ValidatorChunkSize
}

func (c *Config) chunkOffset(epoch uint64) uint64 {
	return epoch % c.ChunkSize
}

func (c *Config) validatorOffset(validatorIndex uint64) uint64 {
	return validatorIndex % c.ValidatorChunkSize
}

// Map the validator and epoch chunk indexes into a single value for use as a database key.
func (c *Config) diskKey(validatorChunkIndex uint64, chunkIndex uint64) uint64 {
	width := c.HistoryLength / c.ChunkSize
	return validatorChunkIndex*width + chunkIndex
}

// Map the validator and epoch offsets into an index for chunk data.
func (c *Config) cellIndex(validatorOffset uint64, chunkOffset uint64) uint64 {
	return validatorOffset*c.ChunkSize + chunkOffset
}

func (c *Config) validatorIndicesInChunk(validatorChunkIdx uint64) []uint64 {
	validatorIndices := make([]uint64, 0)
	low := validatorChunkIdx * c.ValidatorChunkSize
	high := (validatorChunkIdx + 1) * c.ValidatorChunkSize
	for i := low; i < high; i++ {
		validatorIndices = append(validatorIndices, i)
	}
	return validatorIndices
}
