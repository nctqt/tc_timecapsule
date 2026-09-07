package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/nctqt/tc_timecapsule/internal/database"
	"github.com/nctqt/tc_timecapsule/internal/jsonhelp"
)

type RegisterUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AuthResponse struct {
	User  UserResponse `json:"user"`
	Token string       `json:"token"`
}

// helper to generate JWT signed with your secret key
func (cfg *apiConfig) makeJWT(userID uuid.UUID, expiresIn time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		Issuer:    "tc_timecapsule",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		Subject:   userID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.jwtSecret))
}

func (cfg *apiConfig) handlerRegisterUser(w http.ResponseWriter, r *http.Request) {
	var req RegisterUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || !strings.Contains(email, "@") {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "A valid email address is required", nil)
		return
	}

	if len(req.Password) < 8 {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Password must be at least 8 characters long", nil)
		return
	}

	// hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not hash password", err)
		return
	}

	now := time.Now().UTC()
	newUser, err := cfg.queries.CreateUser(r.Context(), database.CreateUserParams{
		ID:             uuid.New(),
		Email:          email,
		HashedPassword: string(hashedPassword),
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		// unique constraint error if email already exists
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			jsonhelp.RespondWithError(w, http.StatusConflict, "An account with this email already exists", err)
			return
		}
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not create user", err)
		return
	}

	// issue JWT token (valid for 24 hours)
	token, err := cfg.makeJWT(newUser.ID, 24*time.Hour)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not issue auth token", err)
		return
	}

	jsonhelp.RespondWithJSON(w, http.StatusCreated, AuthResponse{
		User: UserResponse{
			ID:        newUser.ID,
			Email:     newUser.Email,
			CreatedAt: newUser.CreatedAt,
			UpdatedAt: newUser.UpdatedAt,
		},
		Token: token,
	})
}

func (cfg *apiConfig) handlerLoginUser(w http.ResponseWriter, r *http.Request) {
	var req LoginUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Password == "" {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Email and password are required", nil)
		return
	}

	// fetch user record
	user, err := cfg.queries.GetUserByEmail(r.Context(), email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// avoid revealing whether the email exists or not
			jsonhelp.RespondWithError(w, http.StatusUnauthorized, "Invalid email or password", nil)
			return
		}
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Database error retrieving user", err)
		return
	}

	// verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(req.Password))
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusUnauthorized, "Invalid email or password", nil)
		return
	}

	// issue JWT token (valid for 24 hours)
	token, err := cfg.makeJWT(user.ID, 24*time.Hour)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not issue auth token", err)
		return
	}

	jsonhelp.RespondWithJSON(w, http.StatusOK, AuthResponse{
		User: UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
		Token: token,
	})
}
