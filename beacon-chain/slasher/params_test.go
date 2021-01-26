package slasher

import (
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestDefaultConfig(t *testing.T) {
	def := DefaultParams()
	assert.Equal(t, true, def.ChunkSize > 0)
	assert.Equal(t, true, def.ValidatorChunkSize > 0)
	assert.Equal(t, true, def.HistoryLength > 0)
}

func TestConfig_cellIndex(t *testing.T) {
	type args struct {
		validatorIndex types.ValidatorIdx
		epoch          types.Epoch
	}
	tests := []struct {
		name   string
		fields *Parameters
		args   args
		want   uint64
	}{
		{
			name: "epoch 0 and validator index 0",
			fields: &Parameters{
				ChunkSize:          3,
				ValidatorChunkSize: 3,
			},
			args: args{
				validatorIndex: 0,
				epoch:          0,
			},
			want: 0,
		},
		{
			//     val0     val1     val2
			//      |        |        |
			//   {     }  {     }  {     }
			//  [2, 2, 2, 2, 2, 2, 2, 2, 2]
			//                        |-> epoch 1, validator 2
			name: "epoch < ChunkSize and validatorIndex < ValidatorChunkSize",
			fields: &Parameters{
				ChunkSize:          3,
				ValidatorChunkSize: 3,
			},
			args: args{
				validatorIndex: 2,
				epoch:          1,
			},
			want: 7,
		},
		{
			//     val0     val1     val2
			//      |        |        |
			//   {     }  {     }  {     }
			//  [2, 2, 2, 2, 2, 2, 2, 2, 2]
			//                        |-> epoch 4, validator 2 (wrap around)
			name: "epoch > ChunkSize and validatorIndex < ValidatorChunkSize",
			fields: &Parameters{
				ChunkSize:          3,
				ValidatorChunkSize: 3,
			},
			args: args{
				validatorIndex: 2,
				epoch:          4,
			},
			want: 7,
		},
		{
			//     val0     val1     val2
			//      |        |        |
			//   {     }  {     }  {     }
			//  [2, 2, 2, 2, 2, 2, 2, 2, 2]
			//                     |-> epoch 3, validator 2 (wrap around)
			name: "epoch = ChunkSize and validatorIndex < ValidatorChunkSize",
			fields: &Parameters{
				ChunkSize:          3,
				ValidatorChunkSize: 3,
			},
			args: args{
				validatorIndex: 2,
				epoch:          3,
			},
			want: 6,
		},
		{
			//     val0     val1     val2
			//      |        |        |
			//   {     }  {     }  {     }
			//  [2, 2, 2, 2, 2, 2, 2, 2, 2]
			//   |-> epoch 0, validator 3 (wrap around)
			name: "epoch < ChunkSize and validatorIndex = ValidatorChunkSize",
			fields: &Parameters{
				ChunkSize:          3,
				ValidatorChunkSize: 3,
			},
			args: args{
				validatorIndex: 3,
				epoch:          0,
			},
			want: 0,
		},
		{
			//     val0     val1     val2
			//      |        |        |
			//   {     }  {     }  {     }
			//  [2, 2, 2, 2, 2, 2, 2, 2, 2]
			//            |-> epoch 0, validator 4 (wrap around)
			name: "epoch < ChunkSize and validatorIndex > ValidatorChunkSize",
			fields: &Parameters{
				ChunkSize:          3,
				ValidatorChunkSize: 3,
			},
			args: args{
				validatorIndex: 4,
				epoch:          0,
			},
			want: 3,
		},
		{
			//     val0     val1     val2
			//      |        |        |
			//   {     }  {     }  {     }
			//  [2, 2, 2, 2, 2, 2, 2, 2, 2]
			//   |-> epoch 3, validator 3 (wrap around)
			name: "epoch = ChunkSize and validatorIndex = ValidatorChunkSize",
			fields: &Parameters{
				ChunkSize:          3,
				ValidatorChunkSize: 3,
			},
			args: args{
				validatorIndex: 3,
				epoch:          3,
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Parameters{
				ChunkSize:          tt.fields.ChunkSize,
				ValidatorChunkSize: tt.fields.ValidatorChunkSize,
				HistoryLength:      tt.fields.HistoryLength,
			}
			if got := c.cellIndex(tt.args.validatorIndex, tt.args.epoch); got != tt.want {
				t.Errorf("cellIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_chunkIndex(t *testing.T) {
	tests := []struct {
		name   string
		fields *Parameters
		epoch  types.Epoch
		want   uint64
	}{
		{
			name: "epoch 0",
			fields: &Parameters{
				ChunkSize:     3,
				HistoryLength: 3,
			},
			epoch: 0,
			want:  0,
		},
		{
			name: "epoch < HistoryLength, epoch < ChunkSize",
			fields: &Parameters{
				ChunkSize:     3,
				HistoryLength: 3,
			},
			epoch: 2,
			want:  0,
		},
		{
			name: "epoch = HistoryLength, epoch < ChunkSize",
			fields: &Parameters{
				ChunkSize:     4,
				HistoryLength: 3,
			},
			epoch: 3,
			want:  0,
		},
		{
			name: "epoch > HistoryLength, epoch < ChunkSize",
			fields: &Parameters{
				ChunkSize:     5,
				HistoryLength: 3,
			},
			epoch: 4,
			want:  0,
		},
		{
			name: "epoch < HistoryLength, epoch < ChunkSize",
			fields: &Parameters{
				ChunkSize:     3,
				HistoryLength: 3,
			},
			epoch: 2,
			want:  0,
		},
		{
			name: "epoch = HistoryLength, epoch < ChunkSize",
			fields: &Parameters{
				ChunkSize:     4,
				HistoryLength: 3,
			},
			epoch: 3,
			want:  0,
		},
		{
			name: "epoch < HistoryLength, epoch = ChunkSize",
			fields: &Parameters{
				ChunkSize:     2,
				HistoryLength: 3,
			},
			epoch: 2,
			want:  1,
		},
		{
			name: "epoch < HistoryLength, epoch > ChunkSize",
			fields: &Parameters{
				ChunkSize:     2,
				HistoryLength: 4,
			},
			epoch: 3,
			want:  1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Parameters{
				ChunkSize:     tt.fields.ChunkSize,
				HistoryLength: tt.fields.HistoryLength,
			}
			if got := c.chunkIndex(tt.epoch); got != tt.want {
				t.Errorf("chunkIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_diskKey(t *testing.T) {
	tests := []struct {
		name           string
		fields         *Parameters
		validatorIndex types.ValidatorIdx
		epoch          types.Epoch
		want           uint64
	}{
		{
			name: "Proper disk key for epoch 0, validator 0",
			fields: &Parameters{
				ChunkSize:          3,
				ValidatorChunkSize: 3,
				HistoryLength:      6,
			},
			epoch:          0,
			validatorIndex: 0,
			want:           0,
		},
		{
			name: "Proper disk key for epoch < HistoryLength, validator < ValidatorChunkSize",
			fields: &Parameters{
				ChunkSize:          3,
				ValidatorChunkSize: 3,
				HistoryLength:      6,
			},
			epoch:          1,
			validatorIndex: 1,
			want:           0,
		},
		{
			name: "Proper disk key for epoch > HistoryLength, validator > ValidatorChunkSize",
			fields: &Parameters{
				ChunkSize:          3,
				ValidatorChunkSize: 3,
				HistoryLength:      6,
			},
			epoch:          10,
			validatorIndex: 10,
			want:           7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Parameters{
				ChunkSize:          tt.fields.ChunkSize,
				ValidatorChunkSize: tt.fields.ValidatorChunkSize,
				HistoryLength:      tt.fields.HistoryLength,
			}
			if got := c.flatSliceID(tt.validatorIndex, tt.epoch); got != tt.want {
				t.Errorf("diskKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_validatorChunkIndex(t *testing.T) {
	tests := []struct {
		name           string
		fields         *Parameters
		validatorIndex types.ValidatorIdx
		want           uint64
	}{
		{
			name: "validator index < ValidatorChunkSize",
			fields: &Parameters{
				ValidatorChunkSize: 3,
			},
			validatorIndex: 2,
			want:           0,
		},
		{
			name: "validator index = ValidatorChunkSize",
			fields: &Parameters{
				ValidatorChunkSize: 3,
			},
			validatorIndex: 3,
			want:           1,
		},
		{
			name: "validator index > ValidatorChunkSize",
			fields: &Parameters{
				ValidatorChunkSize: 3,
			},
			validatorIndex: 99,
			want:           33,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Parameters{
				ValidatorChunkSize: tt.fields.ValidatorChunkSize,
			}
			if got := c.validatorChunkIndex(tt.validatorIndex); got != tt.want {
				t.Errorf("validatorChunkIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_chunkOffset(t *testing.T) {
	tests := []struct {
		name   string
		fields *Parameters
		epoch  types.Epoch
		want   uint64
	}{
		{
			name: "epoch < ChunkSize",
			fields: &Parameters{
				ChunkSize: 3,
			},
			epoch: 2,
			want:  2,
		},
		{
			name: "epoch = ChunkSize",
			fields: &Parameters{
				ChunkSize: 3,
			},
			epoch: 3,
			want:  0,
		},
		{
			name: "epoch > ChunkSize",
			fields: &Parameters{
				ChunkSize: 3,
			},
			epoch: 5,
			want:  2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Parameters{
				ChunkSize: tt.fields.ChunkSize,
			}
			if got := c.chunkOffset(tt.epoch); got != tt.want {
				t.Errorf("chunkOffset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_validatorOffset(t *testing.T) {
	tests := []struct {
		name           string
		fields         *Parameters
		validatorIndex types.ValidatorIdx
		want           uint64
	}{
		{
			name: "validatorIndex < ValidatorChunkSize",
			fields: &Parameters{
				ValidatorChunkSize: 3,
			},
			validatorIndex: 2,
			want:           2,
		},
		{
			name: "validatorIndex = ValidatorChunkSize",
			fields: &Parameters{
				ValidatorChunkSize: 3,
			},
			validatorIndex: 3,
			want:           0,
		},
		{
			name: "validatorIndex > ValidatorChunkSize",
			fields: &Parameters{
				ValidatorChunkSize: 3,
			},
			validatorIndex: 5,
			want:           2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Parameters{
				ValidatorChunkSize: tt.fields.ValidatorChunkSize,
			}
			if got := c.validatorOffset(tt.validatorIndex); got != tt.want {
				t.Errorf("validatorOffset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_validatorIndicesInChunk(t *testing.T) {
	tests := []struct {
		name              string
		fields            *Parameters
		validatorChunkIdx uint64
		want              []uint64
	}{
		{
			name: "Returns proper indices",
			fields: &Parameters{
				ValidatorChunkSize: 3,
			},
			validatorChunkIdx: 2,
			want:              []uint64{6, 7, 8},
		},
		{
			name: "0 validator chunk size returs empty",
			fields: &Parameters{
				ValidatorChunkSize: 0,
			},
			validatorChunkIdx: 100,
			want:              []uint64{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Parameters{
				ValidatorChunkSize: tt.fields.ValidatorChunkSize,
			}
			if got := c.validatorIndicesInChunk(tt.validatorChunkIdx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("validatorIndicesInChunk() = %v, want %v", got, tt.want)
			}
		})
	}
}
