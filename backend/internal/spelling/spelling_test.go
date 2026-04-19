package spelling

import "testing"

func TestSyllableSplit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		word string
		want string
	}{
		{name: "caught", word: "caught", want: "c – aught"},
		{name: "because", word: "because", want: "be – cause"},
		{name: "number", word: "number", want: "nu – mber"},
		{name: "music", word: "music", want: "mu – sic"},
		{name: "planet", word: "planet", want: "pla – net"},
		{name: "paper", word: "paper", want: "pa – per"},
		{name: "rabbit", word: "rabbit", want: "ra – bbit"},
		{name: "teacher", word: "teacher", want: "tea – cher"},
		{name: "window", word: "window", want: "wi – ndow"},
		{name: "purple", word: "purple", want: "pu – rple"},
		{name: "empty", word: "", want: ""},
		{name: "single letter", word: "a", want: "a"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := SyllableSplit(tt.word)
			if got != tt.want {
				t.Fatalf("SyllableSplit(%q) = %q, want %q", tt.word, got, tt.want)
			}
		})
	}
}
