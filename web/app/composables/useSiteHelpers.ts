import type { StoredSite } from "~/composables/useSites";

export function siteStatusMeta(status: StoredSite["status"]) {
  switch (status) {
    case "active":
      return { label: "Active", className: "border-primary/30 bg-primary/10 text-primary" };
    case "attention":
      return { label: "Attention", className: "border-accent/30 bg-accent/10 text-accent" };
    case "archived":
      return { label: "Archived", className: "border-border/60 bg-muted/70 text-muted-foreground" };
    default:
      return { label: "Draft", className: "border-sky-500/30 bg-sky-500/10 text-sky-700 dark:text-sky-300" };
  }
}

export function siteDeploymentMeta(state: StoredSite["deployment_state"]) {
  switch (state) {
    case "ready":
      return { label: "Live", className: "border-primary/30 bg-primary/10 text-primary" };
    case "failed":
      return { label: "Failed", className: "border-destructive/30 bg-destructive/10 text-destructive" };
    case "deploying":
      return { label: "Deploying", className: "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-200" };
    default:
      return { label: "Pending", className: "border-border/60 bg-muted/60 text-muted-foreground" };
  }
}

export function siteRuntimeMeta(state: StoredSite["runtime_health_state"]) {
  switch (state) {
    case "healthy":
      return { label: "Healthy", className: "border-primary/30 bg-primary/10 text-primary" };
    case "issue":
      return { label: "Runtime issue", className: "border-destructive/30 bg-destructive/10 text-destructive" };
    case "unknown":
      return { label: "Unknown", className: "border-border/60 bg-muted/60 text-muted-foreground" };
    default:
      return { label: "Checking", className: "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-200" };
  }
}

export function formatSiteDate(iso: string) {
  try {
    return new Date(iso).toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return iso;
  }
}

export function formatRelativeTime(iso: string) {
  try {
    const date = new Date(iso);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return "Just now";
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    return formatSiteDate(iso);
  } catch {
    return iso;
  }
}
