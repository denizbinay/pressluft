package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const SessionCookieName = "pressluft_session"

type Service struct {
	store           *Store
	sessionSecret   []byte
	idleTimeout     time.Duration
	absoluteTimeout time.Duration
	secureCookies   bool
}

func NewService(store *Store, sessionSecret []byte, idleTimeout, absoluteTimeout time.Duration, secureCookies bool) *Service {
	if idleTimeout <= 0 {
		idleTimeout = DefaultSessionIdleTimeout
	}
	if absoluteTimeout <= 0 {
		absoluteTimeout = DefaultSessionAbsoluteTimeout
	}
	return &Service{
		store:           store,
		sessionSecret:   append([]byte(nil), sessionSecret...),
		idleTimeout:     idleTimeout,
		absoluteTimeout: absoluteTimeout,
		secureCookies:   secureCookies,
	}
}

func (s *Service) EnsureBootstrapAdmin(ctx context.Context, email, password string) (*User, error) {
	count, err := s.store.UserCount(ctx)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, nil
	}
	if strings.TrimSpace(email) == "" || strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("bootstrap admin credentials are required")
	}
	return s.store.CreateUser(ctx, email, password, RoleAdmin)
}

func (s *Service) Login(ctx context.Context, w http.ResponseWriter, r *http.Request, email, password string) (Actor, error) {
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return AnonymousActor(), ErrInvalidCredentials
	}
	if err := VerifyPassword(user.PasswordHash, password); err != nil {
		return AnonymousActor(), err
	}
	token, err := GenerateOpaqueToken()
	if err != nil {
		return AnonymousActor(), err
	}
	tokenHash := HashOpaqueToken(s.sessionSecret, token)
	now := time.Now().UTC()
	expiresAt := now.Add(s.idleTimeout)
	absoluteExpiresAt := now.Add(s.absoluteTimeout)
	if absoluteExpiresAt.Before(expiresAt) {
		expiresAt = absoluteExpiresAt
	}
	if err := s.store.CreateSession(ctx, user.ID, tokenHash, expiresAt, absoluteExpiresAt, r.UserAgent(), extractRequestIP(r)); err != nil {
		return AnonymousActor(), err
	}
	if err := s.store.UpdateLastLogin(ctx, user.ID); err != nil {
		return AnonymousActor(), err
	}
	s.setSessionCookie(w, token)
	return userToActor(*user, "session"), nil
}

func (s *Service) Logout(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	token := sessionTokenFromRequest(r)
	if token != "" {
		if err := s.store.RevokeSession(ctx, HashOpaqueToken(s.sessionSecret, token)); err != nil {
			return err
		}
	}
	s.clearSessionCookie(w)
	return nil
}

func (s *Service) AuthenticateRequest(r *http.Request) (Actor, error) {
	token := sessionTokenFromRequest(r)
	if token == "" {
		return AnonymousActor(), ErrUnauthenticated
	}
	hash := HashOpaqueToken(s.sessionSecret, token)
	user, err := s.store.GetSessionUserByHash(r.Context(), hash)
	if err != nil {
		return AnonymousActor(), err
	}
	if err := s.store.TouchSession(r.Context(), hash, time.Now().UTC().Add(s.idleTimeout)); err != nil {
		return AnonymousActor(), err
	}
	return userToActor(*user, "session"), nil
}

func sessionTokenFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

func userToActor(user User, source string) Actor {
	return Actor{
		ID:            fmt.Sprintf("%d", user.ID),
		Type:          ActorTypeOperator,
		Email:         user.Email,
		Role:          user.Role,
		Capabilities:  RoleCapabilities(user.Role),
		Authenticated: true,
		AuthSource:    source,
	}
}

func (s *Service) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.secureCookies,
		MaxAge:   int(s.absoluteTimeout.Seconds()),
	})
}

func (s *Service) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.secureCookies,
		MaxAge:   -1,
	})
}
