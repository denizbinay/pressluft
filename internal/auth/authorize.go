package auth

type Capability string

const (
	CapabilityManageProviders Capability = "manage_providers"
	CapabilityManageServers   Capability = "manage_servers"
	CapabilityQueueJobs       Capability = "queue_jobs"
	CapabilityReadActivity    Capability = "read_activity"
)

func AllCapabilities() []Capability {
	return []Capability{
		CapabilityManageProviders,
		CapabilityManageServers,
		CapabilityQueueJobs,
		CapabilityReadActivity,
	}
}

func RoleCapabilities(role Role) []Capability {
	switch role {
	case RoleAdmin:
		return AllCapabilities()
	default:
		return nil
	}
}

func HasCapability(actor Actor, capability Capability) bool {
	if !actor.IsAuthenticated() {
		return false
	}
	for _, granted := range RoleCapabilities(actor.Role) {
		if granted == capability {
			return true
		}
	}
	return false
}

func RequireCapability(capability Capability) func(Actor) bool {
	return func(actor Actor) bool {
		return HasCapability(actor, capability)
	}
}
