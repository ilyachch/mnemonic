package app

import (
	"github.com/ilyachch/mnemonic/internal/service/catalogsvc"
	"github.com/ilyachch/mnemonic/internal/service/maintsvc"
)

// Services groups the app-owned shared dependencies exposed to adapters.
type Services struct {
	Catalog *catalogsvc.Service
	Maint   *maintsvc.Service
}
