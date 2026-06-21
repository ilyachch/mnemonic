package indexsvc

import "github.com/ilyachch/mnemonic/internal/domain/kb"

// Service is the runtime index service stub.
type Service struct {
	KB kb.KnowledgeBase
}

// New constructs the index service stub for one knowledge base.
func New(kb kb.KnowledgeBase) *Service {
	return &Service{KB: kb}
}
