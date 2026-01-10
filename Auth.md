// Auth contract (dev-only for Weekend 2):
// - Clients must send metadata: "authorization" = "Bearer devtoken"
// - Missing/empty -> codes.Unauthenticated
// - Wrong token   -> codes.PermissionDenied
// - Correct token -> allow RPC