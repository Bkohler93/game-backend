package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/bkohler93/game-backend/pkg/uuidstring"
	"github.com/golang-jwt/jwt/v5"
)

type AuthServer struct {
	addr        string
	rateLimiter *RateLimiter
}

func NewAuthServer(addr string) *AuthServer {
	return &AuthServer{addr, NewRateLimiter()}
}

func (srv *AuthServer) Start(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		log.Println("health check received at \"\\\"")
		_, err := fmt.Fprintln(w, "OK")
		if err != nil {
			log.Println(err)
		}
	})
	mux.HandleFunc("/auth/guest", func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.Header.Get("X-Forwarded-For")
		if clientIP == "" {
			clientIP = r.RemoteAddr
		}
		if srv.rateLimiter.DenyGuestRequest(clientIP) {
			log.Println("denying repeated request for - ", clientIP)
			http.Error(w, "", http.StatusTooManyRequests)
			return
		}

		guestId := uuidstring.NewID()

		claims := jwt.RegisteredClaims{
			// A usual scenario is to set the expiration time relative to the current time
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 10)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "game-backend",
			Subject:   guestId.String(),
			ID:        guestId.String(),
			Audience:  []string{"matchmaker", "setup", "gameplay"},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		jwtStr, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
		if err != nil {
			log.Println("failed to sign token - ", err)
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Length", fmt.Sprint(len(jwtStr)))
		_, err = w.Write([]byte(jwtStr))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", srv.addr),
		Handler: mux,
	}

	go func() {
		log.Println("Starting server on :" + srv.addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server gracefully stopped")
}
