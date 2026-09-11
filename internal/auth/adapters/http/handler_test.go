package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bLorax/khatere-backend/internal/auth/application"
	"github.com/bLorax/khatere-backend/internal/auth/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// --- fakes -------------------------------------------------------

type fakeAccountRepo struct {
	existing *domain.Account
}

func (f *fakeAccountRepo) Create(ctx context.Context, a *domain.Account) error { return nil }
func (f *fakeAccountRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	return nil, domain.ErrAccountNotFound
}
func (f *fakeAccountRepo) FindByEmail(ctx context.Context, email string) (*domain.Account, error) {
	if f.existing != nil && f.existing.Email == email {
		return f.existing, nil
	}
	return nil, domain.ErrAccountNotFound
}
func (f *fakeAccountRepo) Update(ctx context.Context, a *domain.Account) error { return nil }

type fakeHasher struct{}

func (f *fakeHasher) Hash(plaintext string) (string, error) { return "hashed:" + plaintext, nil }
func (f *fakeHasher) Verify(hash, plaintext string) bool    { return hash == "hashed:"+plaintext }

// --- test setup ----------------------------------------------------

func newTestHandlers(repo *fakeAccountRepo) *Handlers {
	registerUC := application.NewRegisterUseCase(repo, &fakeHasher{})
	// login/refresh aren't used by the Register tests below, so nil
	// is fine here — Handlers only calls the use case for the route
	// under test.
	return NewHandlers(registerUC, nil, nil)
}

func doRegisterRequest(h *Handlers, body []byte) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Register(c)
	return w
}

// --- tests ---------------------------------------------------------

func TestHandlers_Register(t *testing.T) {
	t.Run("malformed JSON returns 400", func(t *testing.T) {
		h := newTestHandlers(&fakeAccountRepo{})

		w := doRegisterRequest(h, []byte(`{not json`))

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("existing email returns 409", func(t *testing.T) {
		h := newTestHandlers(&fakeAccountRepo{existing: &domain.Account{Email: "taken@b.com"}})
		body, _ := json.Marshal(map[string]string{
			"email": "taken@b.com", "password": "password123", "account_type": "user",
		})

		w := doRegisterRequest(h, body)

		if w.Code != http.StatusConflict {
			t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
		}
	})

	t.Run("invalid account type returns 400", func(t *testing.T) {
		h := newTestHandlers(&fakeAccountRepo{})
		body, _ := json.Marshal(map[string]string{
			"email": "new@b.com", "password": "password123", "account_type": "wizard",
		})

		w := doRegisterRequest(h, body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
		}
	})

	t.Run("success returns 201 with id and email", func(t *testing.T) {
		h := newTestHandlers(&fakeAccountRepo{})
		body, _ := json.Marshal(map[string]string{
			"email": "new@b.com", "password": "password123", "account_type": "user",
		})

		w := doRegisterRequest(h, body)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("could not parse response body: %v", err)
		}
		if resp["email"] != "new@b.com" {
			t.Errorf("email = %v, want new@b.com", resp["email"])
		}
		if resp["id"] == nil {
			t.Errorf("expected an id in the response")
		}
	})
}
