package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/ilyachch/mnemonic/internal/config"
	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/service/indexsvc"
	"github.com/ilyachch/mnemonic/internal/service/maintsvc"
	"github.com/ilyachch/mnemonic/internal/service/notesvc"
	"github.com/ilyachch/mnemonic/internal/service/searchsvc"
)

// RuntimeInput configures runtime app construction for one knowledge base.
type RuntimeInput struct {
	Config *config.Config
	KB     kb.KnowledgeBase
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
		return nil, fmt.Errorf("knowledge base is required")
	}

	return &RuntimeApp{
		KB: input.KB,
		Services: RuntimeServices{
			Notes:  notesvc.New(input.KB),
			Search: searchsvc.New(input.KB),
			Index:  indexsvc.New(input.KB),
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
func (b *Bootstrap) Runtime(ctx context.Context, selector string) (*RuntimeApp, error) {
	_ = ctx
	if b == nil {
		return nil, fmt.Errorf("app bootstrap is required")
	}
	if b.Services.Catalog == nil {
		return nil, fmt.Errorf("catalog service is not configured")
	}

	resolved, err := b.Services.Catalog.Resolve(selector)
	if err != nil {
		return nil, err
	}

	return NewRuntimeApp(RuntimeInput{
		Config: b.Config,
		KB:     resolved,
	})
}
