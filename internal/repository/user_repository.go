package repository

import (
	"errors"
	"sync"
	"time"

	"github.com/Tnembull/go-core-backend/internal/model"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user with this email already exists")
)

type UserRepository interface {
	Create(user *model.User) error
	FindByEmail(email string) (*model.User, error)
	FindByID(id string) (*model.User, error)
}

type inMemoryUserRepository struct {
	mu    sync.RWMutex
	users map[string]*model.User // email -> user
	byID  map[string]*model.User // id -> user
}

func NewInMemoryUserRepository() UserRepository {
	return &inMemoryUserRepository{
		users: make(map[string]*model.User),
		byID:  make(map[string]*model.User),
	}
}

func (r *inMemoryUserRepository) Create(user *model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.Email]; exists {
		return ErrUserAlreadyExists
	}

	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}

	r.users[user.Email] = user
	r.byID[user.ID] = user
	return nil
}

func (r *inMemoryUserRepository) FindByEmail(email string) (*model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[email]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (r *inMemoryUserRepository) FindByID(id string) (*model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.byID[id]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}
