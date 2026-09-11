package application

import (
	"context"
	"errors"
	"testing"

	"github.com/bLorax/khatere-backend/internal/circle/domain"
	"github.com/google/uuid"
)

// --- fakes -------------------------------------------------------

type fakeConnectionRepo struct {
	createErr  error
	created    *domain.Connection
	createArgs []uuid.UUID // captures [requesterID, addresseeID] for assertions
}

func (f *fakeConnectionRepo) CreateRequest(ctx context.Context, requesterID, addresseeID uuid.UUID) (*domain.Connection, error) {
	f.createArgs = []uuid.UUID{requesterID, addresseeID}
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &domain.Connection{ID: uuid.New(), RequesterID: requesterID, AddresseeID: addresseeID, Status: domain.ConnectionStatusPending}, nil
}
func (f *fakeConnectionRepo) Accept(ctx context.Context, requestID, addresseeID uuid.UUID) (*domain.Connection, error) {
	return nil, nil
}
func (f *fakeConnectionRepo) Decline(ctx context.Context, requestID, addresseeID uuid.UUID) error {
	return nil
}
func (f *fakeConnectionRepo) Sever(ctx context.Context, connectionID, userID uuid.UUID) error {
	return nil
}
func (f *fakeConnectionRepo) ListCircle(ctx context.Context, userID uuid.UUID) ([]domain.Connection, error) {
	return nil, nil
}
func (f *fakeConnectionRepo) ListIncomingPending(ctx context.Context, userID uuid.UUID) ([]domain.Connection, error) {
	return nil, nil
}
func (f *fakeConnectionRepo) ListOutgoingPending(ctx context.Context, userID uuid.UUID) ([]domain.Connection, error) {
	return nil, nil
}

// fakeBlockRepo reports blocked in a given direction based on a
// simple (blocker, blocked) pair match, so a test can express
// "A blocked B" without caring which argument order the use case
// happens to call IsBlocked with.
type fakeBlockRepo struct {
	blockedPairs [][2]uuid.UUID
	isBlockedErr error
}

func (f *fakeBlockRepo) Block(ctx context.Context, blockerID, blockedID uuid.UUID) (*domain.Block, error) {
	return nil, nil
}
func (f *fakeBlockRepo) Unblock(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	return nil
}
func (f *fakeBlockRepo) List(ctx context.Context, blockerID uuid.UUID) ([]domain.Block, error) {
	return nil, nil
}
func (f *fakeBlockRepo) IsBlocked(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error) {
	if f.isBlockedErr != nil {
		return false, f.isBlockedErr
	}
	for _, pair := range f.blockedPairs {
		if pair[0] == blockerID && pair[1] == blockedID {
			return true, nil
		}
	}
	return false, nil
}

// --- tests ---------------------------------------------------------

func TestSendConnectionRequestUseCase_Execute(t *testing.T) {
	requester := uuid.New()
	addressee := uuid.New()

	t.Run("cannot connect to self", func(t *testing.T) {
		uc := NewSendConnectionRequestUseCase(&fakeConnectionRepo{}, &fakeBlockRepo{})

		_, err := uc.Execute(context.Background(), SendConnectionRequestInput{RequesterID: requester, AddresseeID: requester})

		if !errors.Is(err, domain.ErrCannotConnectSelf) {
			t.Errorf("got %v, want ErrCannotConnectSelf", err)
		}
	})

	t.Run("rejected when requester blocked addressee", func(t *testing.T) {
		blocks := &fakeBlockRepo{blockedPairs: [][2]uuid.UUID{{requester, addressee}}}
		uc := NewSendConnectionRequestUseCase(&fakeConnectionRepo{}, blocks)

		_, err := uc.Execute(context.Background(), SendConnectionRequestInput{RequesterID: requester, AddresseeID: addressee})

		if !errors.Is(err, domain.ErrBlocked) {
			t.Errorf("got %v, want ErrBlocked", err)
		}
	})

	t.Run("rejected when addressee blocked requester", func(t *testing.T) {
		blocks := &fakeBlockRepo{blockedPairs: [][2]uuid.UUID{{addressee, requester}}}
		uc := NewSendConnectionRequestUseCase(&fakeConnectionRepo{}, blocks)

		_, err := uc.Execute(context.Background(), SendConnectionRequestInput{RequesterID: requester, AddresseeID: addressee})

		if !errors.Is(err, domain.ErrBlocked) {
			t.Errorf("got %v, want ErrBlocked", err)
		}
	})

	t.Run("block lookup failure is passed through", func(t *testing.T) {
		blocks := &fakeBlockRepo{isBlockedErr: errors.New("db down")}
		uc := NewSendConnectionRequestUseCase(&fakeConnectionRepo{}, blocks)

		_, err := uc.Execute(context.Background(), SendConnectionRequestInput{RequesterID: requester, AddresseeID: addressee})

		if err == nil || err.Error() != "db down" {
			t.Errorf("got %v, want lookup error passed through", err)
		}
	})

	t.Run("duplicate request error from the repo is passed through", func(t *testing.T) {
		connections := &fakeConnectionRepo{createErr: domain.ErrRequestAlreadySent}
		uc := NewSendConnectionRequestUseCase(connections, &fakeBlockRepo{})

		_, err := uc.Execute(context.Background(), SendConnectionRequestInput{RequesterID: requester, AddresseeID: addressee})

		if !errors.Is(err, domain.ErrRequestAlreadySent) {
			t.Errorf("got %v, want ErrRequestAlreadySent", err)
		}
	})

	t.Run("success creates a pending request", func(t *testing.T) {
		connections := &fakeConnectionRepo{}
		uc := NewSendConnectionRequestUseCase(connections, &fakeBlockRepo{})

		got, err := uc.Execute(context.Background(), SendConnectionRequestInput{RequesterID: requester, AddresseeID: addressee})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Status != domain.ConnectionStatusPending {
			t.Errorf("status = %v, want pending", got.Status)
		}
		if connections.createArgs[0] != requester || connections.createArgs[1] != addressee {
			t.Errorf("CreateRequest called with wrong requester/addressee order")
		}
	})
}
