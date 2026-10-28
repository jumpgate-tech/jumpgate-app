// ROUTES lists every "METHOD /pattern" the client calls, in Go ServeMux
// syntax. scripts/check-routes.ts fails CI when one has no Go route.
export const ROUTES = [
  "GET /api/health",
  "GET /api/version",
] as const;

export type Route = (typeof ROUTES)[number];
