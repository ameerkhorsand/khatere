package application

import (
	"context"
	"errors"
	"testing"

	"github.com/bLorax/khatere-backend/internal/auth/domain"
)

func TestRegisterUseCase_Execute(t *testing.T) {
	existingAccount := &domain.Account{Email: "taken@b.com"}

	tests := []struct {
		name    string
		repo    *fakeAccountRepo
		hasher  *fakeHasher
		input   RegisterInput
		wantErr error
	}{
		{
			name:    "invalid account type is rejected",
			repo:    &fakeAccountRepo{},
			hasher:  &fakeHasher{},
			input:   RegisterInput{Email: "a@b.com", Password: "pw", AccountType: "not-a-real-type"},
			wantErr: domain.ErrInvalidAccountType,
		},
		{
			name:    "existing email is rejected",
			repo:    &fakeAccountRepo{account: existingAccount},
			hasher:  &fakeHasher{},
			input:   RegisterInput{Email: "taken@b.com", Password: "pw", AccountType: domain.AccountTypeUser},
			wantErr: domain.ErrEmailAlreadyExists,
		},
		{
			name:    "create failure is passed through",
			repo:    &fakeAccountRepo{findErr: domain.ErrAccountNotFound, createErr: errors.New("db unavailable")},
			hasher:  &fakeHasher{},
			input:   RegisterInput{Email: "new@b.com", Password: "pw", AccountType: domain.AccountTypeUser},
			wantErr: errors.New("db unavailable"),
		},
		{
			name:    "success",
			repo:    &fakeAccountRepo{findErr: domain.ErrAccountNotFound},
			hasher:  &fakeHasher{},
			input:   RegisterInput{Email: "new@b.com", Password: "pw", AccountType: domain.AccountTypeUser},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewRegisterUseCase(tt.repo, tt.hasher)

			account, err := uc.Execute(context.Background(), tt.input)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if account.Email != tt.input.Email {
					t.Errorf("account email = %q, want %q", account.Email, tt.input.Email)
				}
				if account.Version != 1 {
					t.Errorf("new account Version = %d, want 1", account.Version)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if err.Error() != tt.wantErr.Error() {
				t.Errorf("got error %q, want %q", err, tt.wantErr)
			}
		})
	}
}
