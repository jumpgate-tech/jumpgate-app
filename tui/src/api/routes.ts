// ROUTES lists every "METHOD /pattern" the client calls, in Go ServeMux
// syntax. scripts/check-routes.ts fails CI when one has no Go route.
export const ROUTES = [
  "GET /api/health",
  "GET /api/version",
  "GET /api/vault",
  "POST /api/vault/unlock",
  "POST /api/vault/lock",
  "GET /api/fleet",
  "GET /api/fleet/stream",
  "GET /api/fleet/{id}",
  "GET /api/targets",
  "GET /api/fleet/{id}/status/stream",
  "GET /api/fleet/{id}/disk",
  "POST /api/fleet/{id}/disk/measure",
  "GET /api/targets/{id}/endpoints",
  "GET /api/targets/{id}/firewall",
  "GET /api/targets/{id}/logs",
  "GET /api/targets/{id}/logs/stream",
  "POST /api/targets/{id}/services/{svc}/{action}",
  "GET /api/jobs",
  "GET /api/jobs/stream",
  "GET /api/jobs/running",
  "GET /api/jobs/{target}/{job}",
  "GET /api/jobs/{target}/{job}/output/stream",
  "POST /api/jobs/{target}/{job}/decide",
  "POST /api/jobs/{target}/{job}/adopt",
  "GET /api/ui-prefs",
  "PUT /api/ui-prefs",
] as const;

export type Route = (typeof ROUTES)[number];
