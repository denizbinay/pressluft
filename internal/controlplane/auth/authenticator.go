package auth

import "net/http"

type Authenticator interface {
	Authenticate(*http.Request) (Actor, error)
}

type SessionAuthenticator struct {
	service *Service
}

func NewSessionAuthenticator(service *Service) *SessionAuthenticator {
	return &SessionAuthenticator{service: service}
}

func (a *SessionAuthenticator) Authenticate(r *http.Request) (Actor, error) {
	return a.service.AuthenticateRequest(r)
}

type DevAuthenticator struct {
	actor Actor
}

func NewDevAuthenticator() *DevAuthenticator {
	return &DevAuthenticator{
		actor: Actor{
			ID:            "dev-admin",
			Type:          ActorTypeOperator,
			Email:         "dev@example.com",
			Role:          RoleAdmin,
			Capabilities:  RoleCapabilities(RoleAdmin),
			Authenticated: true,
			AuthSource:    "dev",
		},
	}
}

func (a *DevAuthenticator) Authenticate(_ *http.Request) (Actor, error) {
	return a.actor, nil
}
