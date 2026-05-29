export type HealthResponse = {
  ok: boolean;
};

export type PortMapping = {
  protocol: string;
  external_port: number;
  internal_ip: string;
  internal_port: number;
  description: string;
  lease_duration_seconds: number;
};

export type StatusResponse = {
  discovered: boolean;
  service_type?: string;
  control_url?: string;
  external_ip?: string;
  local_ip?: string;
  ports: PortMapping[];
};

export type Settings = {
  listen_addr: string;
  auto_discover: boolean;
};
