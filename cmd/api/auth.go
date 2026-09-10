package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/nctqt/tc_timecapsule/internal/jsonhelp"
)

// authenticateRequest parses the Authorization header and returns the userID if valid.
func (cfg *apiConfig) authenticateRequest(r *http.Request) (uuid.UUID, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return uuid.Nil, fmt.Errorf("authorization header missing")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return uuid.Nil, fmt.Errorf("invalid authorization header format")
	}

	tokenString := parts[1]
	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return uuid.Nil, fmt.Errorf("invalid or expired token: %w", err)
	}

	subject, err := claims.GetSubject()
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid token subject: %w", err)
	}

	userID, err := uuid.Parse(subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID in token: %w", err)
	}

	// db check: verify user still exists in the database
	_, err = cfg.queries.GetUserByID(r.Context(), userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("user account no longer exists")
	}

	return userID, nil
}

// strict auth: rejects the request if no valid token is provided
func (cfg *apiConfig) middlewareAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := cfg.authenticateRequest(r)
		if err != nil {
			jsonhelp.RespondWithError(w, http.StatusUnauthorized, "Unauthorized access", err)
			return
		}

		ctx := context.WithValue(r.Context(), userIDContextKey, userID)
		handler(w, r.WithContext(ctx))
	}
}

// optional auth: passes the request through regardless; attaches userID to context if valid
func (cfg *apiConfig) middlewareOptionalAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := cfg.authenticateRequest(r)
		if err == nil {
			// Token is valid: inject userID into context
			ctx := context.WithValue(r.Context(), userIDContextKey, userID)
			r = r.WithContext(ctx)
		}

		// Always call handler whether authenticated or guest
		handler(w, r)
	}
}

func (cfg *apiConfig) adminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			jsonhelp.RespondWithError(w, http.StatusUnauthorized, "Authorization header missing", nil)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			jsonhelp.RespondWithError(w, http.StatusUnauthorized, "Invalid authorization header format", nil)
			return
		}

		tokenString := parts[1]
		claims := &CustomClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(cfg.jwtSecret), nil
		})

		if err != nil || !token.Valid {
			jsonhelp.RespondWithError(w, http.StatusUnauthorized, "Invalid or expired token", err)
			return
		}

		// Check if the role embedded in the token is admin
		if claims.Role != "admin" {
			jsonhelp.RespondWithError(w, http.StatusForbidden, "Forbidden: Admin access required", nil)
			return
		}

		// Optional: Extract and verify user ID from subject for downstream use
		subject, err := claims.GetSubject()
		if err == nil {
			if userID, err := uuid.Parse(subject); err == nil {
				ctx := context.WithValue(r.Context(), userIDContextKey, userID)
				r = r.WithContext(ctx)
			}
		}

		next(w, r)
	}
}
