package slasher

import (
	"reflect"
	"testing"
)

func TestConfig_cellIndex(t *testing.T) {
	type fields struct {
		ChunkSize          uint64
		ValidatorChunkSize uint64
		HistoryLength      uint64
	}
	type args struct {
		validatorOffset uint64
		chunkOffset     uint64
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
			if got := c.cellIndex(tt.args.validatorOffset, tt.args.chunkOffset); got != tt.want {
				t.Errorf("cellIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_chunkIndex(t *testing.T) {
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
			if got := c.chunkIndex(tt.args.epoch); got != tt.want {
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
			if got := c.validatorChunkIndex(tt.args.validatorIndex); got != tt.want {
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
