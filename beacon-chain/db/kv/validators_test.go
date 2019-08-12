package kv

import (
	"context"
	"reflect"
	"testing"

	"github.com/boltdb/bolt"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestStore_DeleteValidatorIndex(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	tests := []struct {
		name      string
		publicKey [48]byte
		args      args
		wantErr   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := k.DeleteValidatorIndex(tt.args.ctx, tt.args.publicKey); (err != nil) != tt.wantErr {
				t.Errorf("DeleteValidatorIndex() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_HasValidatorIndex(t *testing.T) {
	type fields struct {
		db           *bolt.DB
		DatabasePath string
	}
	type args struct {
		ctx       context.Context
		publicKey [48]byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &Store{
				db:           tt.fields.db,
				DatabasePath: tt.fields.DatabasePath,
			}
			if got := k.HasValidatorIndex(tt.args.ctx, tt.args.publicKey); got != tt.want {
				t.Errorf("HasValidatorIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_HasValidatorLatestVote(t *testing.T) {
	type fields struct {
		db           *bolt.DB
		DatabasePath string
	}
	type args struct {
		ctx          context.Context
		validatorIdx uint64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &Store{
				db:           tt.fields.db,
				DatabasePath: tt.fields.DatabasePath,
			}
			if got := k.HasValidatorLatestVote(tt.args.ctx, tt.args.validatorIdx); got != tt.want {
				t.Errorf("HasValidatorLatestVote() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_SaveValidatorIndex(t *testing.T) {
	type fields struct {
		db           *bolt.DB
		DatabasePath string
	}
	type args struct {
		ctx          context.Context
		publicKey    [48]byte
		validatorIdx uint64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &Store{
				db:           tt.fields.db,
				DatabasePath: tt.fields.DatabasePath,
			}
			if err := k.SaveValidatorIndex(tt.args.ctx, tt.args.publicKey, tt.args.validatorIdx); (err != nil) != tt.wantErr {
				t.Errorf("SaveValidatorIndex() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_SaveValidatorLatestVote(t *testing.T) {
	type fields struct {
		db           *bolt.DB
		DatabasePath string
	}
	type args struct {
		ctx          context.Context
		validatorIdx uint64
		vote         *pb.ValidatorLatestVote
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &Store{
				db:           tt.fields.db,
				DatabasePath: tt.fields.DatabasePath,
			}
			if err := k.SaveValidatorLatestVote(tt.args.ctx, tt.args.validatorIdx, tt.args.vote); (err != nil) != tt.wantErr {
				t.Errorf("SaveValidatorLatestVote() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_ValidatorIndex(t *testing.T) {
	type fields struct {
		db           *bolt.DB
		DatabasePath string
	}
	type args struct {
		ctx       context.Context
		publicKey [48]byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    uint64
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &Store{
				db:           tt.fields.db,
				DatabasePath: tt.fields.DatabasePath,
			}
			got, err := k.ValidatorIndex(tt.args.ctx, tt.args.publicKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatorIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ValidatorIndex() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_ValidatorLatestVote(t *testing.T) {
	type fields struct {
		db           *bolt.DB
		DatabasePath string
	}
	type args struct {
		ctx          context.Context
		validatorIdx uint64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.ValidatorLatestVote
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &Store{
				db:           tt.fields.db,
				DatabasePath: tt.fields.DatabasePath,
			}
			got, err := k.ValidatorLatestVote(tt.args.ctx, tt.args.validatorIdx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatorLatestVote() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidatorLatestVote() got = %v, want %v", got, tt.want)
			}
		})
	}
}
