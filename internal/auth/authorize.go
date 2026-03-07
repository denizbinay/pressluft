package auth

func CanManageProviders(actor Actor) bool {
	return actor.IsAuthenticated() && actor.Role == RoleAdmin
}

func CanManageServers(actor Actor) bool {
	return actor.IsAuthenticated() && actor.Role == RoleAdmin
}

func CanQueueJobs(actor Actor) bool {
	return actor.IsAuthenticated() && actor.Role == RoleAdmin
}

func CanReadActivity(actor Actor) bool {
	return actor.IsAuthenticated() && actor.Role == RoleAdmin
}
