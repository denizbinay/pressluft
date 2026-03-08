package server

import (
	"net/http"

	"pressluft/internal/activity"
	"pressluft/internal/auth"
)

func activityActorFromRequest(r *http.Request) (activity.ActorType, string) {
	actor := auth.ActorFromContext(r.Context())
	if actor.IsAuthenticated() {
		return activity.ActorUser, actor.ID
	}
	return activity.ActorAPI, ""
}
