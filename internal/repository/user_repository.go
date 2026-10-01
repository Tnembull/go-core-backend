package repository

import (
	"errors"
	"sync"
	"time"

	"github.com/Tnembull/go-core-backend/internal/model"
)

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrUserAlreadyExists    = errors.New("user already exists with this email")
	ErrInvalidRefreshToken  = errors.New("invalid or revoked refresh token")
)

type UserRepository interface {
	Create(user *model.User) error
	GetByID(id string) (*model.User, error)
	GetByEmail(email string) (*model.User, error)
	Update(user *model.User) error
	RevokeToken(tokenID string, expiresAt time.Time) error
	IsTokenRevoked(tokenID string) bool
}

type InMemoryUserRepository struct {
	mu           sync.RWMutex
	users        map[string]*model.User // key: user.ID
	usersByEmail map[string]*model.User // key: user.Email
	revokedTokens map[string]time.Time  // key: tokenID/jti, value: expiresAt
}

func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		users:        make(map[string]*model.User),
		usersByEmail: make(map[string]*model.User),
		revokedTokens: make(map[string]time.Time),
	}
}

func (r *InMemoryUserRepository) Create(user *model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.usersByEmail[user.Email]; exists {
		return ErrUserAlreadyExists
	}

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	r.users[user.ID] = user
	r.usersByEmail[user.Email] = user
	return nil
}

func (r *InMemoryUserRepository) GetByID(id string) (*model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (r *InMemoryUserRepository) GetByEmail(email string) (*model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.usersByEmail[email]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (r *InMemoryUserRepository) Update(user *model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.ID]; !exists {
		return ErrUserNotFound
	}

	user.UpdatedAt = time.Now()
	r.users[user.ID] = user
	r.usersByEmail[user.Email] = user
	return nil
}

func (r *InMemoryUserRepository) RevokeToken(tokenID string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.revokedTokens[tokenID] = expiresAt
	return nil
}

func (r *InMemoryUserRepository) IsTokenRevoked(tokenID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	exp, exists := r.revokedTokens[tokenID]
	if !exists {
		return false
	}
	if time.Now().After(exp) {
		return false
	}
	return true
}
