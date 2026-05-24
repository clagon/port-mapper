export type HealthResponse = {
  ok: boolean;
};

export type StatusResponse = {
  discovered: boolean;
  ports: unknown[];
};

export type Settings = {
  listen_addr: string;
  auto_discover: boolean;
};
