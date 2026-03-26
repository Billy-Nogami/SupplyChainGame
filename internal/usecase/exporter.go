package usecase

import (
	"context"

	"supply-chain-simulator/internal/domain"
)

type ExportedFile struct {
	FileName    string
	ContentType string
	Content     []byte
}

type SessionExporter interface {
	ExportSession(ctx context.Context, session domain.GameSession, analytics domain.SessionAnalytics) (ExportedFile, error)
}
