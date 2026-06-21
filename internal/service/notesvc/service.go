package notesvc

import "github.com/ilyachch/mnemonic/internal/domain/kb"

// Service is the runtime notes service stub.
type Service struct {
	KB kb.KnowledgeBase
}

// New constructs the notes service stub for one knowledge base.
func New(kb kb.KnowledgeBase) *Service {
	return &Service{KB: kb}
}
