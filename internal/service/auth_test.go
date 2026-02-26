package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/flicky/go-ecommerce-api/internal/dto"
	"github.com/flicky/go-ecommerce-api/internal/model"
)

type mockUserRepo struct {
	users map[string]*model.User
	byID  map[uuid.UUID]*model.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*model.User), byID: make(map[uuid.UUID]*model.User)}
}

func (m *mockUserRepo) Create(_ context.Context, user *model.User) error {
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.Email] = user
	m.byID[user.ID] = user
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	return m.byID[id], nil
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	return m.users[email], nil
}

func TestAuthService_Register(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, "test-secret", time.Hour)

	resp, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email: "test@example.com", Password: "password123",
		FirstName: "John", LastName: "Doe",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, "test@example.com", resp.User.Email)
	assert.Equal(t, "customer", resp.User.Role)
}

func TestAuthService_Register_Duplicate(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, "test-secret", time.Hour)

	repo.users["test@example.com"] = &model.User{Email: "test@example.com"}

	_, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email: "test@example.com", Password: "password123",
		FirstName: "John", LastName: "Doe",
	})
	assert.ErrorIs(t, err, ErrUserAlreadyExists)
}

func TestAuthService_Login(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, "test-secret", time.Hour)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	repo.users["test@example.com"] = &model.User{
		ID: uuid.New(), Email: "test@example.com", Password: string(hashed), Role: "customer",
	}

	resp, err := svc.Login(context.Background(), dto.LoginRequest{
		Email: "test@example.com", Password: "password123",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, "test-secret", time.Hour)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	repo.users["test@example.com"] = &model.User{
		ID: uuid.New(), Email: "test@example.com", Password: string(hashed),
	}

	_, err := svc.Login(context.Background(), dto.LoginRequest{
		Email: "test@example.com", Password: "wrong",
	})
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}
