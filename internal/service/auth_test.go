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
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*model.User)}
}

func (m *mockUserRepo) Create(_ context.Context, user *model.User) error {
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	return m.users[email], nil
}

func TestAuthService_Register(t *testing.T) {
	svc := NewAuthService(newMockUserRepo(), "secret", time.Hour)
	resp, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email: "test@example.com", Password: "password123",
		FirstName: "John", LastName: "Doe",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, "test@example.com", resp.User.Email)
}

func TestAuthService_Register_Duplicate(t *testing.T) {
	repo := newMockUserRepo()
	repo.users["test@example.com"] = &model.User{Email: "test@example.com"}
	svc := NewAuthService(repo, "secret", time.Hour)
	_, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email: "test@example.com", Password: "password123",
		FirstName: "John", LastName: "Doe",
	})
	assert.ErrorIs(t, err, ErrUserAlreadyExists)
}

func TestAuthService_Login(t *testing.T) {
	repo := newMockUserRepo()
	hashed, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	repo.users["test@example.com"] = &model.User{
		ID: uuid.New(), Email: "test@example.com", Password: string(hashed), Role: "customer",
	}
	svc := NewAuthService(repo, "secret", time.Hour)
	resp, err := svc.Login(context.Background(), dto.LoginRequest{
		Email: "test@example.com", Password: "password123",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	repo := newMockUserRepo()
	hashed, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	repo.users["test@example.com"] = &model.User{
		ID: uuid.New(), Email: "test@example.com", Password: string(hashed),
	}
	svc := NewAuthService(repo, "secret", time.Hour)
	_, err := svc.Login(context.Background(), dto.LoginRequest{
		Email: "test@example.com", Password: "wrong",
	})
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}
