package llm

import "testing"

func TestNewProviderFromEnvDefaultsToMock(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")

	provider := NewProviderFromEnv()
	if provider.Name() != "mock" {
		t.Fatalf("expected mock provider, got %q", provider.Name())
	}
}

func TestNewProviderFromEnvSelectsOpenAI(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "test-openai-key")
	t.Setenv("OPENAI_MODEL", "gpt-4o-mini")

	provider := NewProviderFromEnv()
	if provider.Name() != "openai" {
		t.Fatalf("expected openai provider, got %q", provider.Name())
	}
}

func TestNewProviderFromEnvSelectsGemini(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "gemini")
	t.Setenv("GEMINI_API_KEY", "test-gemini-key")
	t.Setenv("GEMINI_MODEL", "gemini-2.5-flash")

	provider := NewProviderFromEnv()
	if provider.Name() != "gemini" {
		t.Fatalf("expected gemini provider, got %q", provider.Name())
	}
}

func TestNewProviderFromEnvGeminiMissingKeyFallsBackToMock(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "gemini")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")

	provider := NewProviderFromEnv()
	if provider.Name() != "mock" {
		t.Fatalf("expected fallback mock provider, got %q", provider.Name())
	}
}
