package slasher

// Config parameters for slashing detection. Validator min and max spans
// split into chunks of length C for easy detection and lazy loading.
//
// A regular pair of min and max spans for a validator look as follows
// with length = H where H is the amount of epochs worth of history
// we want to persist for slashing detection.
//
//  validator_1_min_span = [2, 2, 2, ..., 2]
//  validator_1_max_span = [0, 0, 0, ..., 0]
//
// After chunking, we obtain chunks of length C. For C = 3:
//
//  validator_1_min_span_chunk_0 = [2, 2, 2]
//  validator_1_max_span_chunk_0 = [2, 2, 2]
//
// On disk, we take chunks for K validators, and store them as flat byte slices.
// For example, if H = 3, C = 3, and K = 3, then we can store 3 validators' chunks as a flat
// slice as follows:
//
//     val0     val1     val2
//      |        |        |
//   {     }  {     }  {     }
//  [2, 2, 2, 2, 2, 2, 2, 2, 2]
//
// This is known as 2D chunking, pioneered by the Sigma Prime team here:
// https://hackmd.io/@sproul/min-max-slasher
//
// To properly access the element at epoch `e` for a validator index `i`, we leverage helper
// functions from config values as a nice abstraction. the following parameters are
// required for the helper functions defined in this file.
//
// ChunkSize defines how many elements are in a chunk for a validator
// min or max span slice.
// ValidatorChunkSize defines how many validators' chunks we store in a single
// flat byte slice on disk.
// HistoryLength defines how many epochs we keep of min or max spans.
type Config struct {
	ChunkSize          uint64
	ValidatorChunkSize uint64
	HistoryLength      uint64
}

// DefaultConfig defines default values for slasher's configuration, defined
// based on optimization analysis for best and worst case scenarios for
// slasher's performance.
func DefaultConfig() *Config {
	return &Config{
		// The default values for ChunkSize and ValidatorChunkSize were
		// decided after an optimization analysis performed by the Sigma Prime team.
		// See: https://hackmd.io/@sproul/min-max-slasher#1D-vs-2D for more information.
		// We decide to keep 4096 epochs worth of data in each validator's min max spans.
		ChunkSize:          16,
		ValidatorChunkSize: 256,
		HistoryLength:      4096,
	}
}

// Validator min and max spans are split into chunks of length C = ChunkSize.
// That is, if we are keeping N epochs worth of attesting history, finding what
// chunk a certain epoch, e, falls into can be computed as (e % N) / C. For example,
// if we are keeping 6 epochs worth of data, and we have chunks of size 2, then epoch
// 4 will fall into chunk index (4 % 6) / 2 = 2.
//
//  span    = [2, 2, 2, 2, 2, 2]
//  chunked = [[2, 2], [2, 2], [2, 2]]
//                              |-> epoch 4, chunk idx 2
func (c *Config) chunkIndex(epoch uint64) uint64 {
	return (epoch % c.HistoryLength) / c.ChunkSize
}

// When storing data on disk, we take K validators' chunks. To figure out
// which validator chunk index a validator index is for, we simply divide
// the validator index, v, by K.
func (c *Config) validatorChunkIndex(validatorIndex uint64) uint64 {
	return validatorIndex / c.ValidatorChunkSize
}

// Given a validator index, and epoch, we compute the exact index
// into our flat slice on disk which stores K validators' chunks, each
// chunk of size C. For example, if C = 3 and K = 3, the data we store
// on disk is a flat slice as follows:
//
//     val0     val1     val2
//      |        |        |
//   {     }  {     }  {     }
//  [2, 2, 2, 2, 2, 2, 2, 2, 2]
//
// Then, figuring out the exact cell index for epoch 1 for validator 2 is computed
// with (validatorIndex % K)*C + (epoch % C), which gives us:
//
//  (2 % 3)*3 + (1 % 3) =
//  (2*3) + (1)         =
//  7
//
//     val0     val1     val2
//      |        |        |
//   {     }  {     }  {     }
//  [2, 2, 2, 2, 2, 2, 2, 2, 2]
//                        |-> epoch 1 for val2
//
func (c *Config) cellIndex(validatorIndex, epoch uint64) uint64 {
	validatorChunkOffset := c.validatorOffset(validatorIndex)
	chunkOffset := c.chunkOffset(epoch)
	return validatorChunkOffset*c.ChunkSize + chunkOffset
}

// Computes the start index of a chunk given an epoch.
func (c *Config) chunkOffset(epoch uint64) uint64 {
	return epoch % c.ChunkSize
}

// Computes the start index of a validator chunk given a validator index.
func (c *Config) validatorOffset(validatorIndex uint64) uint64 {
	return validatorIndex % c.ValidatorChunkSize
}

// Construct a key for our database schema given a validator index and epoch.
// This calculation gives us a uint that uniquely represents
// a 2D chunk given a validator index and epoch value.
// First, we compute the validator chunk index for the validator index,
// Then, we compute the chunk index for the epoch.
// If ChunkSize C = 3 and ValidatorChunkSize K = 3, and HistoryLength H = 12,
// if we are looking for epoch 6 and validator 6, then
//
//  validatorChunkIndex = 6 / 3 = 2
//  chunkIndex = (6 % HistoryLength) / 3 = (6 % 12) / 3 = 2
//
// Then we compute how many chunks there are per max span, known as the "width"
//
//  width = H / C = 12 / 3 = 4
//
// So every span has 4 chunks. Then, we have a disk key calculated by
//
//  validatorChunkIndex * width + chunkIndex = 2*4 + 2 = 10
//
func (c *Config) flatSliceID(validatorIndex, epoch uint64) uint64 {
	validatorChunkIndex := c.validatorChunkIndex(validatorIndex)
	chunkIndex := c.chunkIndex(epoch)
	width := c.HistoryLength / c.ChunkSize
	return validatorChunkIndex*width + chunkIndex
}

// Given a validator chunk index, we determine all of the validator
// indices that will belong in that chunk.
func (c *Config) validatorIndicesInChunk(validatorChunkIdx uint64) []uint64 {
	validatorIndices := make([]uint64, 0)
	low := validatorChunkIdx * c.ValidatorChunkSize
	high := (validatorChunkIdx + 1) * c.ValidatorChunkSize
	for i := low; i < high; i++ {
		validatorIndices = append(validatorIndices, i)
	}
	return validatorIndices
}
