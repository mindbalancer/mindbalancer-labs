package provider

import (
	"time"

	"github.com/mindbalancer/mindbalancer/internal/storage"
)

// Groq implements the Provider interface for Groq.
// Groq uses OpenAI-compatible API, so we extend OpenAI provider.
type Groq struct {
	*OpenAI
}

// NewGroq creates a new Groq provider.
func NewGroq(server storage.Server, timeout time.Duration) *Groq {
	return &Groq{
		OpenAI: NewOpenAI(server, timeout),
	}
}

func (p *Groq) Name() string {
	return "groq"
}

func (p *Groq) SupportsEmbeddings() bool {
	return false
}
