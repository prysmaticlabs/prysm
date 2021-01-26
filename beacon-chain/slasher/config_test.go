package slasher

import (
	"reflect"
	"testing"
)

func TestConfig_cellIndex(t *testing.T) {
	type args struct {
		validatorIndex uint64
		epoch          uint64
	}
	tests := []struct {
		name   string
		fields *Config
		args   args
		want   uint64
	}{
		{
			name: "epoch 0 and validator index 0",
			fields: &Config{
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
			fields: &Config{
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
			fields: &Config{
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
			fields: &Config{
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
			fields: &Config{
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
			fields: &Config{
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
			fields: &Config{
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
			c := &Config{
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
		fields *Config
		epoch  uint64
		want   uint64
	}{
		{
			name: "epoch 0",
			fields: &Config{
				ChunkSize:     3,
				HistoryLength: 3,
			},
			epoch: 0,
			want:  0,
		},
		{
			name: "epoch < HistoryLength, epoch < ChunkSize",
			fields: &Config{
				ChunkSize:     3,
				HistoryLength: 3,
			},
			epoch: 2,
			want:  0,
		},
		{
			name: "epoch = HistoryLength, epoch < ChunkSize",
			fields: &Config{
				ChunkSize:     4,
				HistoryLength: 3,
			},
			epoch: 3,
			want:  0,
		},
		{
			name: "epoch > HistoryLength, epoch < ChunkSize",
			fields: &Config{
				ChunkSize:     5,
				HistoryLength: 3,
			},
			epoch: 4,
			want:  0,
		},
		{
			name: "epoch < HistoryLength, epoch < ChunkSize",
			fields: &Config{
				ChunkSize:     3,
				HistoryLength: 3,
			},
			epoch: 2,
			want:  0,
		},
		{
			name: "epoch = HistoryLength, epoch < ChunkSize",
			fields: &Config{
				ChunkSize:     4,
				HistoryLength: 3,
			},
			epoch: 3,
			want:  0,
		},
		{
			name: "epoch < HistoryLength, epoch = ChunkSize",
			fields: &Config{
				ChunkSize:     2,
				HistoryLength: 3,
			},
			epoch: 2,
			want:  1,
		},
		{
			name: "epoch < HistoryLength, epoch > ChunkSize",
			fields: &Config{
				ChunkSize:     2,
				HistoryLength: 4,
			},
			epoch: 3,
			want:  1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				ChunkSize:     tt.fields.ChunkSize,
				HistoryLength: tt.fields.HistoryLength,
			}
			if got := c.chunkIndex(tt.epoch); got != tt.want {
				t.Errorf("chunkIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_chunkOffset(t *testing.T) {
	type fields struct {
		ChunkSize          uint64
		ValidatorChunkSize uint64
		HistoryLength      uint64
	}
	type args struct {
		epoch uint64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   uint64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				ChunkSize:          tt.fields.ChunkSize,
				ValidatorChunkSize: tt.fields.ValidatorChunkSize,
				HistoryLength:      tt.fields.HistoryLength,
			}
			if got := c.chunkOffset(tt.args.epoch); got != tt.want {
				t.Errorf("chunkOffset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_diskKey(t *testing.T) {
	type fields struct {
		ChunkSize          uint64
		ValidatorChunkSize uint64
		HistoryLength      uint64
	}
	type args struct {
		validatorChunkIndex uint64
		chunkIndex          uint64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   uint64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				ChunkSize:          tt.fields.ChunkSize,
				ValidatorChunkSize: tt.fields.ValidatorChunkSize,
				HistoryLength:      tt.fields.HistoryLength,
			}
			if got := c.diskKey(tt.args.validatorChunkIndex, tt.args.chunkIndex); got != tt.want {
				t.Errorf("diskKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_validatorChunkIndex(t *testing.T) {
	tests := []struct {
		name           string
		fields         *Config
		validatorIndex uint64
		want           uint64
	}{
		{
			name: "validator index < ValidatorChunkSize",
			fields: &Config{
				ValidatorChunkSize: 3,
			},
			validatorIndex: 2,
			want:           0,
		},
		{
			name: "validator index = ValidatorChunkSize",
			fields: &Config{
				ValidatorChunkSize: 3,
			},
			validatorIndex: 3,
			want:           1,
		},
		{
			name: "validator index > ValidatorChunkSize",
			fields: &Config{
				ValidatorChunkSize: 3,
			},
			validatorIndex: 99,
			want:           33,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				ValidatorChunkSize: tt.fields.ValidatorChunkSize,
			}
			if got := c.validatorChunkIndex(tt.validatorIndex); got != tt.want {
				t.Errorf("validatorChunkIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_validatorIndicesInChunk(t *testing.T) {
	type fields struct {
		ChunkSize          uint64
		ValidatorChunkSize uint64
		HistoryLength      uint64
	}
	type args struct {
		validatorChunkIdx uint64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []uint64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				ChunkSize:          tt.fields.ChunkSize,
				ValidatorChunkSize: tt.fields.ValidatorChunkSize,
				HistoryLength:      tt.fields.HistoryLength,
			}
			if got := c.validatorIndicesInChunk(tt.args.validatorChunkIdx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("validatorIndicesInChunk() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_validatorOffset(t *testing.T) {
	type fields struct {
		ChunkSize          uint64
		ValidatorChunkSize uint64
		HistoryLength      uint64
	}
	type args struct {
		validatorIndex uint64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   uint64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				ChunkSize:          tt.fields.ChunkSize,
				ValidatorChunkSize: tt.fields.ValidatorChunkSize,
				HistoryLength:      tt.fields.HistoryLength,
			}
			if got := c.validatorOffset(tt.args.validatorIndex); got != tt.want {
				t.Errorf("validatorOffset() = %v, want %v", got, tt.want)
			}
		})
	}
}
