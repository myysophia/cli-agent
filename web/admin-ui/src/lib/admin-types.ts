export interface AdminHealthResponse {
  status: string;
  time: string;
}

export interface ReleaseNote {
  version: string;
  release_date: string;
  changelog: string;
  url: string;
}

export interface CLIReleaseNotes {
  cli_name: string;
  display_name: string;
  latest_version: string;
  local_version: string;
  update_available: boolean;
  last_updated: string;
  releases: ReleaseNote[];
}

export interface ReleaseNotesResponse {
  clis: Record<string, CLIReleaseNotes>;
  last_updated: string;
}

export interface ChatRequest {
  prompt: string;
  profile?: string;
  cli?: string;
  workflow_run_id?: string;
  [key: string]: unknown;
}

export type ChatResponse = Record<string, unknown>;

export interface AdminConfigMeta {
  exists: boolean;
  source: string;
  last_loaded?: string;
  warnings?: string[];
}

export type GatewayConfig = Record<string, unknown>;

export interface AdminConfigResponse {
  config: GatewayConfig | null;
  meta: AdminConfigMeta;
}

export interface AdminProfileSummary {
  key: string;
  name: string;
  cli?: string;
  model?: string;
  tools_count: number;
  skills_count: number;
  env_count: number;
  is_default: boolean;
}

export interface AdminProfileEnvItem {
  key: string;
  value: string;
  masked: boolean;
}

export interface AdminProfilePayload {
  key: string;
  name: string;
  cli?: string;
  model?: string;
  allowed_tools?: string[];
  skills?: string[];
  system_prompt?: string;
  system_prompt_masked?: boolean;
  env?: AdminProfileEnvItem[];
  is_default?: boolean;
}

export interface AdminProfilesResponse {
  default: string;
  profiles: AdminProfileSummary[];
}

export interface AdminMCPEnvItem {
  key: string;
  value: string;
  masked: boolean;
}

export interface AdminMCPServer {
  name: string;
  command: string;
  args?: string[];
  env?: AdminMCPEnvItem[];
}

export interface AdminMCPMeta {
  path: string;
  display_path: string;
  exists: boolean;
  last_modified?: string;
}

export interface AdminMCPResponse {
  servers: AdminMCPServer[];
  meta: AdminMCPMeta;
}
