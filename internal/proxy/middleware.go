package proxy

// Middleware definitions for the Corvade proxy server.
//
// Currently, header extraction and provider detection are handled inline
// in server.go's handleProxy method. This file is reserved for future
// middleware (e.g., rate limiting, request logging, auth validation)
// as the proxy grows.
