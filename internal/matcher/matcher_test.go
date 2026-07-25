package matcher

import "testing"

func TestMatches(t *testing.T) {
	t.Parallel()

	m := New()
	tests := []struct {
		name    string
		message string
		want    bool
	}{
		{name: "server IP", message: "サーバーIP教えて", want: true},
		{name: "casual Japanese", message: "マイクラって今できる？", want: true},
		{name: "server availability", message: "鯖、開いてる？", want: true},
		{name: "English", message: "Minecraft server status?", want: true},
		{name: "unrelated IP", message: "自分のIPが分からない", want: false},
		{name: "unrelated Minecraft", message: "今日マイクラやろう", want: false},
		{name: "empty", message: "", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := m.Matches(tt.message); got != tt.want {
				t.Fatalf("Matches(%q) = %v, want %v", tt.message, got, tt.want)
			}
		})
	}
}
