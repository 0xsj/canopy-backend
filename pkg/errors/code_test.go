package errors

import "testing"

func TestCode_String(t *testing.T) {
	tests := []struct {
		code Code
		want string
	}{
		{Code("exploration_leaf_not_found"), "exploration_leaf_not_found"},
		{Code("identity_token_expired"), "identity_token_expired"},
		{Code(""), ""},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.code.String(); got != tt.want {
				t.Errorf("Code.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
