package v2

import "testing"

func Test_validatePasswordInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "no numbers nor special characters",
			input:   "abcdefghijklmnopqrs",
			wantErr: true,
		},
		{
			name:    "number and letters but no special characters",
			input:   "abcdefghijklmnopqrs2020",
			wantErr: true,
		},
		{
			name:    "numbers, letters, special characters, but too short",
			input:   "abc2$",
			wantErr: true,
		},
		{
			name:    "proper length and strong password",
			input:   "%Str0ngpassword32kjAjsd22020$%",
			wantErr: false,
		},
		{
			name:    "password format correct but weak entropy score",
			input:   "aaaaaaa1$",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validatePasswordInput(tt.input); (err != nil) != tt.wantErr {
				t.Errorf("validatePasswordInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
