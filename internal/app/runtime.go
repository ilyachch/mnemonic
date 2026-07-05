package app

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/platform/config"
	"github.com/ilyachch/mnemonic/internal/service/indexsvc"
	"github.com/ilyachch/mnemonic/internal/service/maintsvc"
	"github.com/ilyachch/mnemonic/internal/service/notesvc"
	"github.com/ilyachch/mnemonic/internal/service/searchsvc"
)

// RuntimeInput configures runtime app construction for one knowledge base.
type RuntimeInput struct {
	Config *config.Config
	KB     kb.KnowledgeBase
	Logger *slog.Logger
}

// RuntimeServices groups the runtime services for one knowledge base.
type RuntimeServices struct {
	Notes  *notesvc.Service
	Search *searchsvc.Service
	Index  *indexsvc.Service
}

// RuntimeApp is the runtime container for one resolved knowledge base.
type RuntimeApp struct {
	KB       kb.KnowledgeBase
	Services RuntimeServices
}

// NewRuntimeApp builds the runtime container from one resolved knowledge base.
func NewRuntimeApp(input RuntimeInput) (*RuntimeApp, error) {
	if strings.TrimSpace(input.KB.ID) == "" {
		return nil, errors.New("knowledge base is required")
	}

	return &RuntimeApp{
		KB: input.KB,
		Services: RuntimeServices{
			Notes:  notesvc.New(input.KB, input.Logger),
			Search: searchsvc.New(input.KB, input.Logger),
			Index:  indexsvc.New(input.KB, input.Logger),
		},
	}, nil
}

// IndexService returns the runtime index service.
func (r *RuntimeApp) IndexService() maintsvc.IndexService {
	if r == nil {
		return nil
	}
	return r.Services.Index
}

// Runtime resolves a selector into a runtime app.
func (b *Bootstrap) Runtime(ctx context.Context, selector string, logger *slog.Logger) (*RuntimeApp, error) {
	_ = ctx
	if b == nil {
		return nil, errors.New("app bootstrap is required")
	}
	if b.Services.Catalog == nil {
		return nil, errors.New("catalog service is not configured")
	}

	resolved, err := b.Services.Catalog.Resolve(selector, logger)
	if err != nil {
		return nil, err
	}

	return NewRuntimeApp(RuntimeInput{
		Config: b.Config,
		KB:     resolved,
		Logger: logger,
	})
}
