package main

import "testing"

func TestNewGreeting(t *testing.T) {
	tests := []struct {
		name     string
		lang     string
		text     string
		wantLang string
		wantText string
	}{
		{
			name:     "English greeting",
			lang:     "en",
			text:     "Hello",
			wantLang: "en",
			wantText: "Hello",
		},
		{
			name:     "Spanish greeting",
			lang:     "es",
			text:     "Hola",
			wantLang: "es",
			wantText: "Hola",
		},
		{
			name:     "Empty strings",
			lang:     "",
			text:     "",
			wantLang: "",
			wantText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewGreeting(tt.lang, tt.text)
			if got.Language != tt.wantLang {
				t.Errorf("NewGreeting() Language = %v, want %v", got.Language, tt.wantLang)
			}
			if got.Text != tt.wantText {
				t.Errorf("NewGreeting() Text = %v, want %v", got.Text, tt.wantText)
			}
		})
	}
}

func TestGreetingString(t *testing.T) {
	tests := []struct {
		name     string
		greeting Greeting
		want     string
	}{
		{
			name:     "English greeting",
			greeting: Greeting{Language: "en", Text: "Hello"},
			want:     "[en] Hello",
		},
		{
			name:     "Spanish greeting",
			greeting: Greeting{Language: "es", Text: "Hola"},
			want:     "[es] Hola",
		},
		{
			name:     "French greeting",
			greeting: Greeting{Language: "fr", Text: "Bonjour"},
			want:     "[fr] Bonjour",
		},
		{
			name:     "Empty values",
			greeting: Greeting{Language: "", Text: ""},
			want:     "[] ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.greeting.String()
			if got != tt.want {
				t.Errorf("Greeting.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
