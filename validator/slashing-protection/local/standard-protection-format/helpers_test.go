package interchangeformat

import (
	"reflect"
	"testing"
)

func Test_uint64FromString(t *testing.T) {
	tests := []struct {
		name    string
		str     string
		want    uint64
		wantErr bool
	}{
		{
			name:    "Overflow uint64 fails",
			str:     "2934890283904829038490283904829038490",
			wantErr: true,
		},
		{
			name:    "Negative number fails",
			str:     "-3",
			wantErr: true,
		},
		{
			name:    "Junk fails",
			str:     "alksjdkjasd",
			wantErr: true,
		},
		{
			name: "0 works",
			str:  "0",
			want: 0,
		},
		{
			name: "Normal uint64 works",
			str:  "23980",
			want: 23980,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := uint64FromString(tt.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("uint64FromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("uint64FromString() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_pubKeyFromHex(t *testing.T) {
	tests := []struct {
		name    string
		str     string
		want    []byte
		wantErr bool
	}{
		{
			name:    "Empty value fails due to wrong length",
			str:     "",
			wantErr: true,
		},
		{
			name:    "Junk fails",
			str:     "alksjdkjasd",
			wantErr: true,
		},
		{
			name:    "Empty value with 0x prefix fails due to wrong length",
			str:     "0x",
			wantErr: true,
		},
		{
			name:    "Works with 0x prefix and good public key",
			str:     "0xb845089a1457f811bfc000588fbb4e713669be8ce060ea6be3c6ece09afc3794106c91ca73acda5e5457122d58723bed",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pubKeyFromHex(tt.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("pubKeyFromHex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("pubKeyFromHex() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_rootFromHex(t *testing.T) {
	tests := []struct {
		name    string
		str     string
		want    []byte
		wantErr bool
	}{
		//{
		//	name: "Works without 0x prefix and good public key",
		//	str:  "b845089a1457f811bfc000588fbb4e713669be8ce060ea6be3c6ece09afc3794106c91ca73acda5e5457122d58723bed",
		//	want: [32]byte{},
		//},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rootFromHex(tt.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("rootFromHex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rootFromHex() got = %v, want %v", got, tt.want)
			}
		})
	}
}
