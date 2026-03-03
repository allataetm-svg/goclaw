package provider

import (
	"context"
	"fmt"
	"time"
)

type FallbackProvider struct {
	primary   LLMProvider
	fallbacks []LLMProvider
	config    RetryConfig
}

type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
}

func NewFallbackProvider(primary LLMProvider, fallbacks []LLMProvider, retry RetryConfig) *FallbackProvider {
	return &FallbackProvider{
		primary:   primary,
		fallbacks: fallbacks,
		config:    retry,
	}
}

func (p *FallbackProvider) ID() string   { return p.primary.ID() }
func (p *FallbackProvider) Name() string { return p.primary.Name() }

func (p *FallbackProvider) FetchModels() ([]string, error) {
	models, err := p.primary.FetchModels()
	if err != nil && len(p.fallbacks) > 0 {
		return p.fallbacks[0].FetchModels()
	}
	return models, err
}

func (p *FallbackProvider) Query(ctx context.Context, model string, messages []ChatMessage) (string, error) {
	return p.queryWithRetry(ctx, model, messages, 0)
}

func (p *FallbackProvider) queryWithRetry(ctx context.Context, model string, messages []ChatMessage, attempt int) (string, error) {
	result, err := p.primary.Query(ctx, model, messages)
	if err == nil {
		return result, nil
	}

	if attempt < p.config.MaxRetries {
		delay := p.config.BaseDelay * time.Duration(attempt+1)
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(delay):
			return p.queryWithRetry(ctx, model, messages, attempt+1)
		}
	}

	for _, fb := range p.fallbacks {
		result, err := fb.Query(ctx, model, messages)
		if err == nil {
			return result, nil
		}
	}

	return "", fmt.Errorf("all providers failed: last error: %w", err)
}

func (p *FallbackProvider) QueryStream(ctx context.Context, model string, messages []ChatMessage, ch chan<- StreamChunk) {
	p.queryStreamWithRetry(ctx, model, messages, ch, 0)
}

func (p *FallbackProvider) queryStreamWithRetry(ctx context.Context, model string, messages []ChatMessage, ch chan<- StreamChunk, attempt int) {
	done := make(chan struct{})
	var err error
	var result string

	go func() {
		result, err = p.primary.Query(ctx, model, messages)
		close(done)
	}()

	select {
	case <-done:
		if err == nil {
			ch <- StreamChunk{Text: result, Done: true}
			return
		}

		if attempt < p.config.MaxRetries {
			delay := p.config.BaseDelay * time.Duration(attempt+1)
			select {
			case <-ctx.Done():
				ch <- StreamChunk{Error: ctx.Err()}
				return
			case <-time.After(delay):
				p.queryStreamWithRetry(ctx, model, messages, ch, attempt+1)
				return
			}
		}

		for _, fb := range p.fallbacks {
			fbCh := make(chan StreamChunk)
			go fb.QueryStream(ctx, model, messages, fbCh)
			for chunk := range fbCh {
				if chunk.Error != nil {
					continue
				}
				ch <- chunk
			}
			return
		}

		ch <- StreamChunk{Error: fmt.Errorf("all providers failed: last error: %w", err)}
	case <-ctx.Done():
		ch <- StreamChunk{Error: ctx.Err()}
	}
}
