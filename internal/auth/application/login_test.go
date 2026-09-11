package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bLorax/khatere-backend/internal/auth/domain"
	"github.com/google/uuid"
)

// --- fakes -------------------------------------------------------

type fakeAccountRepo struct {
	account   *domain.Account
	findErr   error
	createErr error // add this line
}

func (f *fakeAccountRepo) Create(ctx context.Context, a *domain.Account) error { return f.createErr } // change this line
func (f *fakeAccountRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	return f.account, f.findErr
}
func (f *fakeAccountRepo) FindByEmail(ctx context.Context, email string) (*domain.Account, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	return f.account, nil
}
func (f *fakeAccountRepo) Update(ctx context.Context, a *domain.Account) error { return nil }

type fakeRefreshTokenRepo struct {
	createErr error
}

func (f *fakeRefreshTokenRepo) Create(ctx context.Context, t *domain.RefreshToken) error {
	return f.createErr
}
func (f *fakeRefreshTokenRepo) FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	return nil, nil
}
func (f *fakeRefreshTokenRepo) Revoke(ctx context.Context, id uuid.UUID) error { return nil }

type fakeHasher struct {
	verifyResult bool
}

func (f *fakeHasher) Hash(plaintext string) (string, error) { return "hashed", nil }
func (f *fakeHasher) Verify(hash, plaintext string) bool    { return f.verifyResult }

type fakeTokenService struct {
	issueErr error
}

func (f *fakeTokenService) IssueAccessToken(id uuid.UUID, t domain.AccountType) (string, error) {
	if f.issueErr != nil {
		return "", f.issueErr
	}
	return "access-token", nil
}
func (f *fakeTokenService) GenerateRefreshTokenValue() (string, error) { return "refresh-plain", nil }
func (f *fakeTokenService) HashRefreshToken(plaintext string) string   { return "refresh-hash" }

// --- tests ---------------------------------------------------------

func TestLoginUseCase_Execute(t *testing.T) {
	validAccount := &domain.Account{
		ID:           uuid.New(),
		PasswordHash: "hashed-pw",
		AccountType:  domain.AccountTypeUser,
	}
	deletedTime := time.Now()
	deletedAccount := &domain.Account{
		ID:           uuid.New(),
		PasswordHash: "hashed-pw",
		DeletedAt:    &deletedTime,
	}

	tests := []struct {
		name        string
		accountRepo *fakeAccountRepo
		hasher      *fakeHasher
		tokens      *fakeTokenService
		wantErr     error
	}{
		{
			name:        "account not found maps to invalid credentials",
			accountRepo: &fakeAccountRepo{findErr: domain.ErrAccountNotFound},
			hasher:      &fakeHasher{verifyResult: true},
			tokens:      &fakeTokenService{},
			wantErr:     domain.ErrInvalidCredentials,
		},
		{
			name:        "deleted account is rejected",
			accountRepo: &fakeAccountRepo{account: deletedAccount},
			hasher:      &fakeHasher{verifyResult: true},
			tokens:      &fakeTokenService{},
			wantErr:     domain.ErrInvalidCredentials,
		},
		{
			name:        "wrong password is rejected",
			accountRepo: &fakeAccountRepo{account: validAccount},
			hasher:      &fakeHasher{verifyResult: false},
			tokens:      &fakeTokenService{},
			wantErr:     domain.ErrInvalidCredentials,
		},
		{
			name:        "token issue failure is passed through",
			accountRepo: &fakeAccountRepo{account: validAccount},
			hasher:      &fakeHasher{verifyResult: true},
			tokens:      &fakeTokenService{issueErr: errors.New("signing key unavailable")},
			wantErr:     errors.New("signing key unavailable"),
		},
		{
			name:        "success",
			accountRepo: &fakeAccountRepo{account: validAccount},
			hasher:      &fakeHasher{verifyResult: true},
			tokens:      &fakeTokenService{},
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewLoginUseCase(
				tt.accountRepo,
				&fakeRefreshTokenRepo{},
				tt.hasher,
				tt.tokens,
				time.Hour,
			)

			out, err := uc.Execute(context.Background(), LoginInput{
				Email:    "a@b.com",
				Password: "whatever",
			})

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if out.AccessToken == "" || out.RefreshToken == "" {
					t.Errorf("expected tokens to be set on success")
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error %v, got nil", tt.wantErr)
			}
			if errors.Is(err, domain.ErrInvalidCredentials) && !errors.Is(tt.wantErr, domain.ErrInvalidCredentials) {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}
