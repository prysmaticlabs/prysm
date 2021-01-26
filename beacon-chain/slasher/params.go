package slasher

// Parameters for slashing detection.
//
// To properly access the element at epoch `e` for a validator index `i`, we leverage helper
// functions from these parameter values as nice abstractions. the following parameters are
// required for the helper functions defined in this file.
//
// (C) ChunkSize defines how many elements are in a chunk for a validator
// min or max span slice.
// (K) ValidatorChunkSize defines how many validators' chunks we store in a single
// flat byte slice on disk.
// (H) HistoryLength defines how many epochs we keep of min or max spans.
type Parameters struct {
	ChunkSize          uint64
	ValidatorChunkSize uint64
	HistoryLength      uint64
}

// DefaultParams defines default values for slasher's important parameters, defined
// based on optimization analysis for best and worst case scenarios for
// slasher's performance.
func DefaultParams() *Parameters {
	return &Parameters{
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
func (p *Parameters) chunkIndex(epoch uint64) uint64 {
	return (epoch % p.HistoryLength) / p.ChunkSize
}

// When storing data on disk, we take K validators' chunks. To figure out
// which validator chunk index a validator index is for, we simply divide
// the validator index, i, by K.
func (p *Parameters) validatorChunkIndex(validatorIndex uint64) uint64 {
	return validatorIndex / p.ValidatorChunkSize
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
func (p *Parameters) cellIndex(validatorIndex, epoch uint64) uint64 {
	validatorChunkOffset := p.validatorOffset(validatorIndex)
	chunkOffset := p.chunkOffset(epoch)
	return validatorChunkOffset*p.ChunkSize + chunkOffset
}

// Computes the start index of a chunk given an epoch.
func (p *Parameters) chunkOffset(epoch uint64) uint64 {
	return epoch % p.ChunkSize
}

// Computes the start index of a validator chunk given a validator index.
func (p *Parameters) validatorOffset(validatorIndex uint64) uint64 {
	return validatorIndex % p.ValidatorChunkSize
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
func (p *Parameters) flatSliceID(validatorIndex, epoch uint64) uint64 {
	validatorChunkIndex := p.validatorChunkIndex(validatorIndex)
	chunkIndex := p.chunkIndex(epoch)
	width := p.HistoryLength / p.ChunkSize
	return validatorChunkIndex*width + chunkIndex
}

// Given a validator chunk index, we determine all of the validator
// indices that will belong in that chunk.
func (p *Parameters) validatorIndicesInChunk(validatorChunkIdx uint64) []uint64 {
	validatorIndices := make([]uint64, 0)
	low := validatorChunkIdx * p.ValidatorChunkSize
	high := (validatorChunkIdx + 1) * p.ValidatorChunkSize
	for i := low; i < high; i++ {
		validatorIndices = append(validatorIndices, i)
	}
	return validatorIndices
}
