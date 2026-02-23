package profiles

// Profile describes a server provisioning profile that maps to auditable
// infrastructure artifacts in the infra/profiles directory.
type Profile struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	ArtifactPath string `json:"artifact_path"`
}

var registry = []Profile{
	{
		Key:          "nginx-stack",
		Name:         "NGINX Stack",
		Description:  "Balanced stack for managed WordPress sites with NGINX, PHP-FPM, and baseline hardening.",
		ArtifactPath: "infra/profiles/nginx-stack/profile.yaml",
	},
	{
		Key:          "openlitespeed-stack",
		Name:         "OpenLiteSpeed Stack",
		Description:  "High-performance stack tuned for caching-heavy agency workloads.",
		ArtifactPath: "infra/profiles/openlitespeed-stack/profile.yaml",
	},
	{
		Key:          "woocommerce-optimized",
		Name:         "WooCommerce Optimized",
		Description:  "Commerce-focused profile with queue, cache, and PHP tuning defaults for WooCommerce.",
		ArtifactPath: "infra/profiles/woocommerce-optimized/profile.yaml",
	},
}

// All returns all available server profiles.
func All() []Profile {
	out := make([]Profile, len(registry))
	copy(out, registry)
	return out
}

// Get returns a profile by key.
func Get(key string) (Profile, bool) {
	for _, profile := range registry {
		if profile.Key == key {
			return profile, true
		}
	}
	return Profile{}, false
}
