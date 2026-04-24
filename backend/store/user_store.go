package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"kyc/backend/models"
)

// UserStore handles database operations for users.
type UserStore struct {
	db *DB
}

// NewUserStore creates a new UserStore.
func NewUserStore(db *DB) *UserStore {
	return &UserStore{db: db}
}

// CreateUser inserts a new user and returns the created user.
func (s *UserStore) CreateUser(ctx context.Context, email, hashedPassword, role string) (*models.User, error) {
	user := &models.User{}
	err := s.db.Pool.QueryRow(ctx,
		`INSERT INTO users (email, password, role) VALUES ($1, $2, $3)
		 RETURNING id, email, password, role, created_at`,
		email, hashedPassword, role,
	).Scan(&user.ID, &user.Email, &user.Password, &user.Role, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return user, nil
}

// GetUserByEmail retrieves a user by email address.
func (s *UserStore) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}
	err := s.db.Pool.QueryRow(ctx,
		`SELECT id, email, password, role, created_at FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.Password, &user.Role, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return user, nil
}

// GetUserByID retrieves a user by ID.
func (s *UserStore) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user := &models.User{}
	err := s.db.Pool.QueryRow(ctx,
		`SELECT id, email, password, role, created_at FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Email, &user.Password, &user.Role, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return user, nil
}

// UserExists checks if a user with the given email already exists.
func (s *UserStore) UserExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := s.db.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`,
		email,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}
	return exists, nil
}
