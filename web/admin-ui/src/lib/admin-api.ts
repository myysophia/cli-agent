import { getAdminToken } from "./admin-token";
import type {
  AdminConfigResponse,
  AdminHealthResponse,
  AdminProfilePayload,
  AdminProfilesResponse,
  AdminMCPResponse,
  AdminMCPServer,
  ChatRequest,
  ChatResponse,
  GatewayConfig,
  ReleaseNotesResponse
} from "./admin-types";

const ADMIN_BASE_PATH = "/v1/admin";

function apiUrl(path: string) {
  if (typeof window === "undefined") {
    return `${ADMIN_BASE_PATH}${path}`;
  }
  return `${window.location.origin}${ADMIN_BASE_PATH}${path}`;
}

export async function fetchAdminHealth(): Promise<AdminHealthResponse> {
  const token = getAdminToken();
  const response = await fetch(apiUrl("/health"), {
    headers: {
      Authorization: token ? `Bearer ${token}` : ""
    }
  });

  if (!response.ok) {
    throw new Error(`health request failed: ${response.status}`);
  }

  return response.json() as Promise<AdminHealthResponse>;
}

export async function fetchReleaseNotes(): Promise<ReleaseNotesResponse> {
  const response = await fetch(`${window.location.origin}/release-notes`);
  if (!response.ok) {
    throw new Error(`release notes request failed: ${response.status}`);
  }
  return response.json() as Promise<ReleaseNotesResponse>;
}

export async function callChat(payload: ChatRequest): Promise<ChatResponse> {
  const response = await fetch(`${window.location.origin}/chat`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify(payload)
  });

  if (!response.ok) {
    const body = await response.text();
    throw new Error(body || `chat request failed: ${response.status}`);
  }

  return response.json() as Promise<ChatResponse>;
}

export async function fetchAdminConfig(): Promise<AdminConfigResponse> {
  const token = getAdminToken();
  const response = await fetch(apiUrl("/api/config"), {
    headers: {
      Authorization: token ? `Bearer ${token}` : ""
    }
  });
  if (!response.ok) {
    throw new Error(`config request failed: ${response.status}`);
  }
  return response.json() as Promise<AdminConfigResponse>;
}

export async function saveAdminConfig(payload: GatewayConfig): Promise<AdminConfigResponse> {
  const token = getAdminToken();
  const response = await fetch(apiUrl("/api/config"), {
    method: "POST",
    headers: {
      Authorization: token ? `Bearer ${token}` : "",
      "Content-Type": "application/json"
    },
    body: JSON.stringify(payload)
  });
  if (!response.ok) {
    const body = await response.text();
    throw new Error(body || `config update failed: ${response.status}`);
  }
  return response.json() as Promise<AdminConfigResponse>;
}

export async function reloadAdminConfig(): Promise<AdminConfigResponse> {
  const token = getAdminToken();
  const response = await fetch(apiUrl("/api/config/reload"), {
    method: "POST",
    headers: {
      Authorization: token ? `Bearer ${token}` : ""
    }
  });
  if (!response.ok) {
    const body = await response.text();
    throw new Error(body || `config reload failed: ${response.status}`);
  }
  return response.json() as Promise<AdminConfigResponse>;
}

export async function fetchProfiles(): Promise<AdminProfilesResponse> {
  const token = getAdminToken();
  const response = await fetch(apiUrl("/api/config/profiles"), {
    headers: {
      Authorization: token ? `Bearer ${token}` : ""
    }
  });
  if (!response.ok) {
    throw new Error(`profiles request failed: ${response.status}`);
  }
  return response.json() as Promise<AdminProfilesResponse>;
}

export async function fetchProfile(key: string): Promise<AdminProfilePayload> {
  const token = getAdminToken();
  const response = await fetch(apiUrl(`/api/config/profiles/${encodeURIComponent(key)}`), {
    headers: {
      Authorization: token ? `Bearer ${token}` : ""
    }
  });
  if (!response.ok) {
    throw new Error(`profile request failed: ${response.status}`);
  }
  return response.json() as Promise<AdminProfilePayload>;
}

export async function createProfile(payload: AdminProfilePayload): Promise<AdminProfilePayload> {
  const token = getAdminToken();
  const response = await fetch(apiUrl("/api/config/profiles"), {
    method: "POST",
    headers: {
      Authorization: token ? `Bearer ${token}` : "",
      "Content-Type": "application/json"
    },
    body: JSON.stringify(payload)
  });
  if (!response.ok) {
    const body = await response.text();
    throw new Error(body || `profile create failed: ${response.status}`);
  }
  return response.json() as Promise<AdminProfilePayload>;
}

export async function updateProfile(
  key: string,
  payload: AdminProfilePayload
): Promise<AdminProfilePayload> {
  const token = getAdminToken();
  const response = await fetch(apiUrl(`/api/config/profiles/${encodeURIComponent(key)}`), {
    method: "PUT",
    headers: {
      Authorization: token ? `Bearer ${token}` : "",
      "Content-Type": "application/json"
    },
    body: JSON.stringify(payload)
  });
  if (!response.ok) {
    const body = await response.text();
    throw new Error(body || `profile update failed: ${response.status}`);
  }
  return response.json() as Promise<AdminProfilePayload>;
}

export async function deleteProfile(key: string): Promise<void> {
  const token = getAdminToken();
  const response = await fetch(apiUrl(`/api/config/profiles/${encodeURIComponent(key)}`), {
    method: "DELETE",
    headers: {
      Authorization: token ? `Bearer ${token}` : ""
    }
  });
  if (!response.ok) {
    const body = await response.text();
    throw new Error(body || `profile delete failed: ${response.status}`);
  }
}

export async function fetchCursorMCP(): Promise<AdminMCPResponse> {
  const token = getAdminToken();
  const response = await fetch(apiUrl("/api/mcp/cursor"), {
    headers: {
      Authorization: token ? `Bearer ${token}` : ""
    }
  });
  if (!response.ok) {
    throw new Error(`mcp request failed: ${response.status}`);
  }
  return response.json() as Promise<AdminMCPResponse>;
}

export async function createCursorMCP(server: AdminMCPServer): Promise<AdminMCPServer> {
  const token = getAdminToken();
  const response = await fetch(apiUrl("/api/mcp/cursor"), {
    method: "POST",
    headers: {
      Authorization: token ? `Bearer ${token}` : "",
      "Content-Type": "application/json"
    },
    body: JSON.stringify(server)
  });
  if (!response.ok) {
    const body = await response.text();
    throw new Error(body || `mcp create failed: ${response.status}`);
  }
  return response.json() as Promise<AdminMCPServer>;
}

export async function updateCursorMCP(
  name: string,
  server: AdminMCPServer
): Promise<AdminMCPServer> {
  const token = getAdminToken();
  const response = await fetch(apiUrl(`/api/mcp/cursor/${encodeURIComponent(name)}`), {
    method: "PUT",
    headers: {
      Authorization: token ? `Bearer ${token}` : "",
      "Content-Type": "application/json"
    },
    body: JSON.stringify(server)
  });
  if (!response.ok) {
    const body = await response.text();
    throw new Error(body || `mcp update failed: ${response.status}`);
  }
  return response.json() as Promise<AdminMCPServer>;
}

export async function deleteCursorMCP(name: string): Promise<void> {
  const token = getAdminToken();
  const response = await fetch(apiUrl(`/api/mcp/cursor/${encodeURIComponent(name)}`), {
    method: "DELETE",
    headers: {
      Authorization: token ? `Bearer ${token}` : ""
    }
  });
  if (!response.ok) {
    const body = await response.text();
    throw new Error(body || `mcp delete failed: ${response.status}`);
  }
}

export async function fetchClaudeMCP(): Promise<AdminMCPResponse> {
  const token = getAdminToken();
  const response = await fetch(apiUrl("/api/mcp/claude"), {
    headers: {
      Authorization: token ? `Bearer ${token}` : ""
    }
  });
  if (!response.ok) {
    throw new Error(`mcp request failed: ${response.status}`);
  }
  return response.json() as Promise<AdminMCPResponse>;
}

export async function createClaudeMCP(server: AdminMCPServer): Promise<AdminMCPServer> {
  const token = getAdminToken();
  const response = await fetch(apiUrl("/api/mcp/claude"), {
    method: "POST",
    headers: {
      Authorization: token ? `Bearer ${token}` : "",
      "Content-Type": "application/json"
    },
    body: JSON.stringify(server)
  });
  if (!response.ok) {
    const body = await response.text();
    throw new Error(body || `mcp create failed: ${response.status}`);
  }
  return response.json() as Promise<AdminMCPServer>;
}

export async function updateClaudeMCP(
  name: string,
  server: AdminMCPServer
): Promise<AdminMCPServer> {
  const token = getAdminToken();
  const response = await fetch(apiUrl(`/api/mcp/claude/${encodeURIComponent(name)}`), {
    method: "PUT",
    headers: {
      Authorization: token ? `Bearer ${token}` : "",
      "Content-Type": "application/json"
    },
    body: JSON.stringify(server)
  });
  if (!response.ok) {
    const body = await response.text();
    throw new Error(body || `mcp update failed: ${response.status}`);
  }
  return response.json() as Promise<AdminMCPServer>;
}

export async function deleteClaudeMCP(name: string): Promise<void> {
  const token = getAdminToken();
  const response = await fetch(apiUrl(`/api/mcp/claude/${encodeURIComponent(name)}`), {
    method: "DELETE",
    headers: {
      Authorization: token ? `Bearer ${token}` : ""
    }
  });
  if (!response.ok) {
    const body = await response.text();
    throw new Error(body || `mcp delete failed: ${response.status}`);
  }
}
