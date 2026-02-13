package main

import (
	"context"

	expdomain "github.com/0xsj/canopy-backend/internal/exploration/domain"
	identitydomain "github.com/0xsj/canopy-backend/internal/identity/domain"
	orgsvc "github.com/0xsj/canopy-backend/internal/organization/service"
	sessiondomain "github.com/0xsj/canopy-backend/internal/session/domain"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// userReaderAdapter wraps the identity UserRepository to satisfy the
// organization service's UserReader cross-context port.
type userReaderAdapter struct {
	repo identitydomain.UserRepository
}

func (a *userReaderAdapter) FindByID(ctx context.Context, id types.UserID) (orgsvc.UserInfo, error) {
	user, err := a.repo.FindByID(ctx, id)
	if err != nil {
		return orgsvc.UserInfo{}, err
	}
	return orgsvc.UserInfo{
		ID:          user.ID(),
		DisplayName: user.DisplayName(),
		Email:       user.Email(),
	}, nil
}

// leafWriterAdapter wraps the exploration LeafRepository to satisfy the
// convergence service's LeafWriter cross-context port (string → Layer).
type leafWriterAdapter struct {
	repo expdomain.LeafRepository
}

func (a *leafWriterAdapter) UpdateLayer(ctx context.Context, id types.LeafID, layer string) error {
	return a.repo.UpdateLayer(ctx, id, expdomain.Layer(layer))
}

// noopContextAssembler is a stub for the session service's ContextAssembler
// port. Returns the session's existing messages as-is until the LLM
// integration is built.
type noopContextAssembler struct{}

func (noopContextAssembler) Assemble(_ context.Context, session sessiondomain.Session) ([]sessiondomain.Message, error) {
	return session.Messages(), nil
}
