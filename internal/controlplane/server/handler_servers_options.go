package server

import (
	"net/http"

	"pressluft/internal/controlplane/apitypes"
	"pressluft/internal/controlplane/server/profiles"
	"pressluft/internal/infra/provider"
)

func (sh *serversHandler) handleCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	providerID, err := parseProviderIDQuery(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	storedProvider, err := sh.providerStore.GetByID(r.Context(), providerID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	serverProvider, ok := provider.GetServerProvider(storedProvider.Type)
	if !ok {
		respondError(w, http.StatusBadRequest, "provider does not support server provisioning: "+storedProvider.Type)
		return
	}

	catalog, err := serverProvider.ListServerCatalog(r.Context(), storedProvider.APIToken)
	if err != nil {
		respondError(w, http.StatusBadGateway, "failed to fetch provider server catalog: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, apitypes.ServerCatalogResponse{
		Catalog:  *catalog,
		Profiles: profiles.All(),
	})
}

func (sh *serversHandler) handleRebuildOptions(w http.ResponseWriter, r *http.Request, serverID string) {
	server, err := sh.serverStore.GetByID(r.Context(), serverID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	storedProvider, err := sh.providerStore.GetByID(r.Context(), server.ProviderID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	serverProvider, ok := provider.GetServerProvider(storedProvider.Type)
	if !ok {
		respondError(w, http.StatusBadRequest, "provider does not support server provisioning: "+storedProvider.Type)
		return
	}
	imageProvider, ok := provider.GetServerImageProvider(storedProvider.Type)
	if !ok {
		respondError(w, http.StatusBadRequest, "provider does not support image listing: "+storedProvider.Type)
		return
	}

	catalog, err := serverProvider.ListServerCatalog(r.Context(), storedProvider.APIToken)
	if err != nil {
		respondError(w, http.StatusBadGateway, "failed to fetch provider server catalog: "+err.Error())
		return
	}
	architecture, err := resolveServerArchitecture(server.ServerType, catalog)
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}

	images, err := imageProvider.ListServerImages(r.Context(), storedProvider.APIToken, architecture)
	if err != nil {
		respondError(w, http.StatusBadGateway, "failed to fetch provider images: "+err.Error())
		return
	}
	if images == nil {
		images = []provider.ServerImageOption{}
	}

	respondJSON(w, http.StatusOK, apitypes.RebuildOptionsResponse{
		ServerID:     apitypes.FormatAppID(server.ID),
		ServerType:   server.ServerType,
		Architecture: architecture,
		Images:       images,
	})
}

func (sh *serversHandler) handleResizeOptions(w http.ResponseWriter, r *http.Request, serverID string) {
	server, err := sh.serverStore.GetByID(r.Context(), serverID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	storedProvider, err := sh.providerStore.GetByID(r.Context(), server.ProviderID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	serverProvider, ok := provider.GetServerProvider(storedProvider.Type)
	if !ok {
		respondError(w, http.StatusBadRequest, "provider does not support server provisioning: "+storedProvider.Type)
		return
	}

	catalog, err := serverProvider.ListServerCatalog(r.Context(), storedProvider.APIToken)
	if err != nil {
		respondError(w, http.StatusBadGateway, "failed to fetch provider server catalog: "+err.Error())
		return
	}
	architecture, err := resolveServerArchitecture(server.ServerType, catalog)
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}

	serverTypes := filterServerTypes(catalog.ServerTypes, server.Location, architecture)
	if serverTypes == nil {
		serverTypes = []provider.ServerTypeOption{}
	}

	respondJSON(w, http.StatusOK, apitypes.ResizeOptionsResponse{
		ServerID:     apitypes.FormatAppID(server.ID),
		Location:     server.Location,
		ServerType:   server.ServerType,
		Architecture: architecture,
		ServerTypes:  serverTypes,
	})
}

func (sh *serversHandler) handleServerFirewalls(w http.ResponseWriter, r *http.Request, serverID string) {
	server, err := sh.serverStore.GetByID(r.Context(), serverID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	storedProvider, err := sh.providerStore.GetByID(r.Context(), server.ProviderID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	firewallProvider, ok := provider.GetFirewallProvider(storedProvider.Type)
	if !ok {
		respondError(w, http.StatusBadRequest, "provider does not support firewall listing: "+storedProvider.Type)
		return
	}

	firewalls, err := firewallProvider.ListFirewalls(r.Context(), storedProvider.APIToken)
	if err != nil {
		respondError(w, http.StatusBadGateway, "failed to fetch provider firewalls: "+err.Error())
		return
	}
	if firewalls == nil {
		firewalls = []provider.FirewallOption{}
	}

	respondJSON(w, http.StatusOK, apitypes.FirewallsResponse{
		ServerID:  apitypes.FormatAppID(server.ID),
		Firewalls: firewalls,
	})
}

func (sh *serversHandler) handleServerVolumes(w http.ResponseWriter, r *http.Request, serverID string) {
	server, err := sh.serverStore.GetByID(r.Context(), serverID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	storedProvider, err := sh.providerStore.GetByID(r.Context(), server.ProviderID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	volumeProvider, ok := provider.GetVolumeProvider(storedProvider.Type)
	if !ok {
		respondError(w, http.StatusBadRequest, "provider does not support volume listing: "+storedProvider.Type)
		return
	}

	volumes, err := volumeProvider.ListVolumes(r.Context(), storedProvider.APIToken)
	if err != nil {
		respondError(w, http.StatusBadGateway, "failed to fetch provider volumes: "+err.Error())
		return
	}
	if volumes == nil {
		volumes = []provider.VolumeOption{}
	}

	respondJSON(w, http.StatusOK, apitypes.VolumesResponse{
		ServerID: apitypes.FormatAppID(server.ID),
		Volumes:  volumes,
	})
}
