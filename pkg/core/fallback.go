package core

import (
	"context"
)

// FallbackRunner tries providers in order until one successfully streams a
// completion. This implements the spec's "Avtomatik Zaxira" (seamless fallback):
// if a provider returns 429, a network error, or a session refresh, the request
// is transparently retried on the next provider in the chain instead of failing
// back to the caller.
type FallbackRunner struct {
	registry *Registry
	order    []string
}

// NewFallbackRunner creates a runner that tries providers in the given order.
func NewFallbackRunner(registry *Registry, order []string) *FallbackRunner {
	return &FallbackRunner{registry: registry, order: order}
}

// Run attempts ChatStream across providers until one yields a terminal chunk
// (FinishReason != "") or its stream closes cleanly. Providers that error before
// producing a final chunk are skipped and the next provider in the chain is tried.
// If every provider fails, the last error is sent on outErr.
func (r *FallbackRunner) Run(ctx context.Context, req ChatRequest) (<-chan ChatChunk, <-chan error) {
	outChunk := make(chan ChatChunk, 32)
	outErr := make(chan error, 1)

	go func() {
		defer close(outChunk)
		defer close(outErr)

		var lastErr error
		for _, name := range r.order {
			p, ok := r.registry.Provider(name)
			if !ok {
				continue
			}

			chunkCh, errCh := p.ChatStream(ctx, req)
			streamDone := false
			providerErr := false

			for !streamDone && !providerErr {
				select {
				case <-ctx.Done():
					outErr <- ctx.Err()
					return
				case chunk, ok := <-chunkCh:
					if !ok {
						chunkCh = nil
						streamDone = true
						continue
					}
					outChunk <- chunk
					if chunk.FinishReason != "" {
						streamDone = true
						continue
					}
				case err, ok := <-errCh:
					if !ok {
						errCh = nil
						if chunkCh == nil {
							streamDone = true
						}
						continue
					}
					if err != nil {
						lastErr = err
						providerErr = true
					}
				}
			}

			if streamDone && !providerErr {
				return // Completed successfully.
			}
			// Otherwise try the next provider in the chain.
		}

		if lastErr != nil {
			outErr <- lastErr
		}
	}()

	return outChunk, outErr
}
