import { ref, readonly } from "vue";

export interface ImageOption {
  value: string;
  label: string;
  id?: number;
  name?: string;
  type?: string;
  architecture?: string;
  deprecated?: boolean;
  status?: string;
}

export interface ResizeServerTypeOption {
  value: string;
  label: string;
  cores?: number;
  memory_gb?: number;
  disk_gb?: number;
  architecture?: string;
}

export interface FirewallOption {
  value: string;
  label: string;
  id?: number;
  name?: string;
}

export interface VolumeOption {
  value: string;
  label: string;
  id?: number;
  name?: string;
  size_gb?: number;
  location?: string;
  status?: string;
  server_id?: number;
}

// Raw response shapes returned by the API
interface RawImage {
  id?: number;
  name?: string;
  type?: string;
  architecture?: string;
  deprecated?: boolean;
  status?: string;
}

interface RawServerType {
  name?: string;
  cores?: number;
  memory_gb?: number;
  disk_gb?: number;
  architecture?: string;
}

interface RawFirewall {
  id?: number;
  name?: string;
}

interface RawVolume {
  id?: number;
  name?: string;
  size_gb?: number;
  location?: string;
  status?: string;
  server_id?: number;
}

interface RebuildOptionsResponse {
  images?: RawImage[];
}

interface ResizeOptionsResponse {
  server_types?: RawServerType[];
}

interface FirewallsResponse {
  firewalls?: RawFirewall[];
}

interface VolumesResponse {
  volumes?: RawVolume[];
}

export function useServerOptions() {
  const { apiFetch } = useApiClient();
  const images = ref<ImageOption[]>([]);
  const serverTypes = ref<ResizeServerTypeOption[]>([]);
  const firewalls = ref<FirewallOption[]>([]);
  const volumes = ref<VolumeOption[]>([]);
  const loading = ref(false);
  const error = ref("");

  const fetchServerImages = async (serverId: string) => {
    const body = await apiFetch<RebuildOptionsResponse>(
      `/servers/${serverId}/rebuild-options`,
    );
    const list = Array.isArray(body?.images) ? body.images : [];
    images.value = list
      .map((image: RawImage) => {
        const name = String(image?.name || "").trim();
        const id = Number(image?.id);
        const value = name || (Number.isFinite(id) ? String(id) : "");
        if (!value) return null;
        const deprecated = Boolean(image?.deprecated);
        const label = deprecated ? `${value} (deprecated)` : value;
        return {
          value,
          label,
          id: Number.isFinite(id) ? id : undefined,
          name: name || undefined,
          type: image?.type ? String(image.type) : undefined,
          architecture: image?.architecture
            ? String(image.architecture)
            : undefined,
          deprecated,
          status: image?.status ? String(image.status) : undefined,
        };
      })
      .filter(Boolean) as ImageOption[];
  };

  const fetchServerTypes = async (serverId: string) => {
    const body = await apiFetch<ResizeOptionsResponse>(
      `/servers/${serverId}/resize-options`,
    );
    const list = Array.isArray(body?.server_types) ? body.server_types : [];
    serverTypes.value = list
      .map((type_: RawServerType) => {
        const name = String(type_?.name || "").trim();
        if (!name) return null;
        const cores = Number(type_?.cores);
        const memory = Number(type_?.memory_gb);
        const disk = Number(type_?.disk_gb);
        const detail = [
          Number.isFinite(cores) ? `${cores} vCPU` : null,
          Number.isFinite(memory) ? `${memory}GB RAM` : null,
          Number.isFinite(disk) ? `${disk}GB SSD` : null,
        ]
          .filter(Boolean)
          .join(" · ");
        return {
          value: name,
          label: detail ? `${name} (${detail})` : name,
          cores: Number.isFinite(cores) ? cores : undefined,
          memory_gb: Number.isFinite(memory) ? memory : undefined,
          disk_gb: Number.isFinite(disk) ? disk : undefined,
          architecture: type_?.architecture
            ? String(type_.architecture)
            : undefined,
        };
      })
      .filter(Boolean) as ResizeServerTypeOption[];
  };

  const fetchServerFirewalls = async (serverId: string) => {
    const body = await apiFetch<FirewallsResponse>(
      `/servers/${serverId}/firewalls`,
    );
    const list = Array.isArray(body?.firewalls) ? body.firewalls : [];
    firewalls.value = list
      .map((fw: RawFirewall) => {
        const id = Number(fw?.id);
        const name = String(fw?.name || "").trim();
        if (!name && !Number.isFinite(id)) return null;
        const label =
          name && Number.isFinite(id) ? `${name} (#${id})` : name || String(id);
        return {
          value: Number.isFinite(id) ? String(id) : name,
          label,
          id: Number.isFinite(id) ? id : undefined,
          name: name || undefined,
        };
      })
      .filter(Boolean) as FirewallOption[];
  };

  const fetchServerVolumes = async (serverId: string) => {
    const body = await apiFetch<VolumesResponse>(
      `/servers/${serverId}/volumes`,
    );
    const list = Array.isArray(body?.volumes) ? body.volumes : [];
    volumes.value = list
      .map((vol: RawVolume) => {
        const name = String(vol?.name || "").trim();
        const id = Number(vol?.id);
        if (!name) return null;
        const size = Number(vol?.size_gb);
        const location = vol?.location ? String(vol.location) : "";
        const labelParts = [
          name || (Number.isFinite(id) ? `volume-${id}` : ""),
          Number.isFinite(size) ? `${size}GB` : "",
          location,
        ].filter(Boolean);
        return {
          value: name,
          label: labelParts.join(" · "),
          id: Number.isFinite(id) ? id : undefined,
          name: name || undefined,
          size_gb: Number.isFinite(size) ? size : undefined,
          location: location || undefined,
          status: vol?.status ? String(vol.status) : undefined,
          server_id: Number.isFinite(Number(vol?.server_id))
            ? Number(vol.server_id)
            : undefined,
        };
      })
      .filter(Boolean) as VolumeOption[];
  };

  const fetchAll = async (serverId: string) => {
    loading.value = true;
    error.value = "";
    const results = await Promise.allSettled([
      fetchServerImages(serverId),
      fetchServerTypes(serverId),
      fetchServerFirewalls(serverId),
      fetchServerVolumes(serverId),
    ]);
    const rejected = results.find((result) => result.status === "rejected") as
      | PromiseRejectedResult
      | undefined;
    if (rejected) {
      error.value = rejected.reason?.message || "Failed to load server options";
    }
    loading.value = false;
  };

  return {
    images: readonly(images),
    serverTypes: readonly(serverTypes),
    firewalls: readonly(firewalls),
    volumes: readonly(volumes),
    loading: readonly(loading),
    error: readonly(error),
    fetchServerImages,
    fetchServerTypes,
    fetchServerFirewalls,
    fetchServerVolumes,
    fetchAll,
  };
}
