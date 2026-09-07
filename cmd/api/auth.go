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

func (cfg *apiConfig) middlewareAuth(handler http.HandlerFunc) http.HandlerFunc {
	// return a function in the shape of our handler functions
	return func(w http.ResponseWriter, r *http.Request) {
		// auth header required
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			jsonhelp.RespondWithError(w, http.StatusUnauthorized, "Authorization header required", nil)
			return
		}

		// expect format: "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			jsonhelp.RespondWithError(w, http.StatusUnauthorized, "Invalid authorization header format", nil)
			return
		}

		tokenString := parts[1]

		claims := &jwt.RegisteredClaims{}
		// here is where I lose it
		// passing a func makes it a callback, blocking execution in ParseWithClaims 'a synchronous callback'
		// perhaps better thought of as a key function
		// a function is used for customization, our use case here is simple so it looks ridiculous
		// but the option exists to use this function to support different signing methods or multi-tenancy or rotating keys
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
			// validate HMAC signing method
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(cfg.jwtSecret), nil
		})

		if err != nil || !token.Valid {
			jsonhelp.RespondWithError(w, http.StatusUnauthorized, "Invalid or expired token", err)
			return
		}

		subject, err := claims.GetSubject()
		if err != nil {
			jsonhelp.RespondWithError(w, http.StatusUnauthorized, "Invalid token subject", err)
			return
		}

		userID, err := uuid.Parse(subject)
		if err != nil {
			jsonhelp.RespondWithError(w, http.StatusUnauthorized, "Invalid user ID in token", err)
			return
		}

		// inject userID into request context and pass to the next handler
		ctx := context.WithValue(r.Context(), userIDContextKey, userID)
		handler(w, r.WithContext(ctx))
	}
}
