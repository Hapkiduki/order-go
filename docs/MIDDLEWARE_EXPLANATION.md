# Middleware Explanation

This document explains what each middleware does in the order they are executed in the application.

## 📋 Execution Order

Middlewares execute in the order they are added. The order matters because each one may depend on what the previous one did.

```
Request → [1] RealIP → [2] RequestID → [3] Logger → [4] Recoverer → 
[5] Timeout → [6] CORS → [7] RateLimiter → [8] SecureHeaders → 
[9] APIVersion → [10] ContentTypeJSON → Handler
```

---

## 🔍 Detailed Middlewares

### 1. **RealIP** - Real IP Extraction

**Location**: `middleware.RealIP`

**What it does:**
- Extracts the real client IP when there are proxies/load balancers in front
- Handles multiple comma-separated IPs in `X-Forwarded-For` by taking the first one (original client IP)
- Stores the real IP in request context instead of modifying `RemoteAddr`
- Falls back to `X-Real-IP` header if `X-Forwarded-For` is not present

**Why first?**
- Other middlewares (RateLimiter, Logger) need the real client IP
- If behind a proxy, `r.RemoteAddr` would be the proxy IP, not the client's

**Security Note**: 
- `X-Forwarded-For` can be spoofed by attackers
- In production, validate against trusted proxy IPs
- The first IP in `X-Forwarded-For` is typically the original client IP

**Example**:
```go
// X-Forwarded-For: "203.0.113.1, 192.168.1.1, 10.0.0.1"
// → Takes first IP: "203.0.113.1"
// → Stores in context, doesn't modify RemoteAddr

// Access via GetRealIP(r) helper function
realIP := middleware.GetRealIP(r)  // "203.0.113.1"
```

**Code**:
```go
// Splits X-Forwarded-For by comma and takes first IP
ips := strings.Split(xff, ",")
realIP := strings.TrimSpace(ips[0])

// Stores in context instead of modifying RemoteAddr
ctx := context.WithValue(r.Context(), RealIPKey, realIP)
```

---

### 2. **RequestID** - Request ID Generation

**Location**: `middleware.RequestID`

**What it does:**
- Generates a unique UUID for each request
- If the request already has an `X-Request-ID` header, it reuses it
- Adds the ID to the context and response header

**Why is it important?**
- Allows tracking a specific request through logs
- Useful for debugging and troubleshooting
- Facilitates log correlation in distributed systems

**Example**:
```go
// Request without ID → Generates: "550e8400-e29b-41d4-a716-446655440000"
// Request with X-Request-ID: "abc-123" → Reuses: "abc-123"

// Adds to context:
ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

// Adds to response header:
w.Header().Set("X-Request-ID", requestID)
```

**Usage in logs**:
```json
{
  "level": "info",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "POST",
  "path": "/api/v1/orders"
}
```

---

### 3. **Logger** - HTTP Request Logging

**Location**: `middleware.Logger(logger)`

**What it does:**
- Logs each HTTP request with complete details
- Captures: method, path, query params, status code, latency, IP, user agent
- Uses the RequestID from context (that's why it goes after RequestID)

**Information logged**:
- `request_id`: Unique request ID
- `method`: GET, POST, PUT, DELETE, etc.
- `path`: Request path (`/api/v1/orders`)
- `query`: Query parameters (`?status=pending`)
- `status`: HTTP response code (200, 404, 500, etc.)
- `latency_ms`: Processing time in milliseconds
- `client_ip`: Client IP (already processed by RealIP)
- `user_agent`: Browser/client that made the request

**Example log**:
```json
{
  "level": "info",
  "msg": "HTTP Request",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "POST",
  "path": "/api/v1/orders",
  "query": "",
  "status": 201,
  "latency_ms": 45,
  "client_ip": "203.0.113.1",
  "user_agent": "Mozilla/5.0..."
}
```

**Technique**: Uses a `responseWriter` wrapper to capture the status code before the response is written.

---

### 4. **Recoverer** - Panic Handling

**Location**: `middleware.Recoverer(logger)`

**What it does:**
- Captures panics that occur during request processing
- Prevents the server from crashing due to a panic
- Logs the error and stack trace
- Returns a 500 JSON error to the client

**Why is it important?**
- Prevents server crashes
- Provides useful information for debugging (stack trace)
- Gives a controlled response to the client instead of a closed connection

**Example**:
```go
// If a panic occurs:
defer func() {
    if err := recover(); err != nil {
        logger.Error("Panic recovered",
            "request_id", requestID,
            "error", err,
            "path", r.URL.Path,
            "stack", string(debug.Stack()),
        )
        
        // Returns JSON error
        w.WriteHeader(500)
        w.Write(`{"success":false,"error":{"code":"INTERNAL_ERROR"}}`)
    }
}()
```

**Without Recoverer**: Server crashes
**With Recoverer**: Error is logged and controlled response is returned

---

### 5. **Timeout** - Request Timeout

**Location**: `chimiddleware.Timeout(30 * time.Second)` (from Chi)

**What it does:**
- Sets a maximum timeout for each request (30 seconds by default)
- If the request takes longer, it cancels it and returns 504 Gateway Timeout
- Prevents slow requests from consuming resources indefinitely

**Why is it important?**
- Protects against hanging requests
- Frees resources (goroutines, DB connections) after timeout
- Improves user experience (doesn't wait indefinitely)

**Example**:
```go
// Request that takes 35 seconds:
// → Canceled after 30 seconds
// → Returns: {"success":false,"error":{"code":"TIMEOUT"}}
// → Status: 504 Gateway Timeout
```

**Note**: This middleware comes from Chi, it's not custom. Uses `context.WithTimeout`.

---

### 6. **CORS** - Cross-Origin Resource Sharing

**Location**: `cors.Handler(cors.Options{...})` (from Chi)

**What it does:**
- Handles cross-origin requests
- Adds appropriate CORS headers
- Handles preflight requests (OPTIONS)

**Current configuration**:
- `AllowedOrigins`: Allowed origins (configured in config)
- `AllowedMethods`: GET, POST, PUT, PATCH, DELETE, OPTIONS
- `AllowedHeaders`: Accept, Authorization, Content-Type, X-Request-ID
- `ExposedHeaders`: X-Request-ID, X-API-Version
- `AllowCredentials`: true (allows cookies/auth)
- `MaxAge`: 300 seconds (preflight cache)

**Example**:
```http
Request:
Origin: https://example.com
Access-Control-Request-Method: POST

Response:
Access-Control-Allow-Origin: https://example.com
Access-Control-Allow-Methods: POST, GET, PUT, DELETE
Access-Control-Allow-Headers: Content-Type, Authorization
Access-Control-Max-Age: 300
```

---

### 7. **RateLimiter** - Rate Limiting

**Location**: `middleware.RateLimiter(config)`

**What it does:**
- Limits the number of requests per second per client
- Uses Token Bucket algorithm
- Default: 10 requests/second, burst of 20

**Default configuration**:
- `RequestsPerSecond`: 10
- `Burst`: 20
- `KeyFunc`: Uses client IP (`r.RemoteAddr`)

**How it works**:
1. Each client has its own "bucket" of tokens
2. Each request consumes a token
3. Tokens regenerate at a fixed rate (10/second)
4. If no tokens available → 429 Too Many Requests

**Example**:
```go
// Client makes 25 rapid requests:
// → First 20: OK (burst)
// → Requests 21-25: 429 Too Many Requests
// → After 1 second: Can make 10 more
```

**Response when exceeded**:
```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMITED",
    "message": "Too many requests, please try again later"
  }
}
```
Status: `429 Too Many Requests`
Header: `Retry-After: 1`

**Note**: Uses a thread-safe map with `sync.RWMutex` to store limiters per client.

---

### 8. **SecureHeaders** - Security Headers

**Location**: `middleware.SecureHeaders`

**What it does:**
- Adds HTTP security headers to all responses
- Protects against various types of common attacks

**Headers added**:

1. **X-Content-Type-Options: nosniff**
   - Prevents browser from "guessing" MIME type
   - Protects against MIME type sniffing attacks

2. **X-Frame-Options: DENY**
   - Prevents page from being displayed in an iframe
   - Protects against clickjacking

3. **X-XSS-Protection: 1; mode=block**
   - Enables browser XSS filter
   - Blocks suspicious content

4. **Strict-Transport-Security: max-age=31536000; includeSubDomains**
   - Forces HTTPS for 1 year
   - Prevents downgrade attacks

5. **Content-Security-Policy: default-src 'self'**
   - Restricts where resources can be loaded from
   - Prevents XSS and data injection

6. **Referrer-Policy: strict-origin-when-cross-origin**
   - Controls what referrer information is sent
   - Protects privacy

**Example response**:
```http
HTTP/1.1 200 OK
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'self'
Referrer-Policy: strict-origin-when-cross-origin
```

---

### 9. **APIVersion** - API Version

**Location**: `middleware.APIVersion(version)`

**What it does:**
- Adds `X-API-Version` header to all responses
- Indicates which API version is being used

**Example**:
```http
HTTP/1.1 200 OK
X-API-Version: 1.0.0
```

**Why is it useful?**
- Clients can verify which version they're using
- Useful for debugging and support
- Facilitates version migration

**Value**: Passed from `main.go` (can be "dev", "1.0.0", etc.)

---

### 10. **ContentTypeJSON** - Content-Type Validation

**Location**: `middleware.ContentTypeJSON`

**What it does:**
- Validates that write requests (POST, PUT, PATCH) have a valid JSON content type
- Accepts `application/json` and variants with charset parameters (e.g., `application/json; charset=utf-8`)
- Empty Content-Type is allowed (will be set to application/json in response)
- If Content-Type is present but not valid JSON, returns 415 Unsupported Media Type
- Always sets `Content-Type: application/json` in responses

**Validation**:
```go
// For POST, PUT, PATCH:
// Uses mime.ParseMediaType() to handle charset parameters
// Accepts: "application/json", "application/json; charset=utf-8", etc.
// Rejects: "text/plain", "application/xml", etc.
// Allows: empty Content-Type
```

**Example error**:
```json
// Request with Content-Type: text/plain
// → Response:
{
  "success": false,
  "error": {
    "code": "UNSUPPORTED_MEDIA_TYPE",
    "message": "Content-Type must be application/json"
  }
}
```
Status: `415 Unsupported Media Type`

**Accepted Content-Types**:
- `application/json` ✅
- `application/json; charset=utf-8` ✅
- `application/json; charset=UTF-8` ✅
- (empty) ✅
- `text/plain` ❌
- `application/xml` ❌

**Responses**: Always sets `Content-Type: application/json` in all responses.

---

## 🔄 Complete Request Flow

```
1. Client sends request
   ↓
2. RealIP: Extracts real client IP
   ↓
3. RequestID: Generates/obtains unique ID
   ↓
4. Logger: Logs request start
   ↓
5. Recoverer: Prepares panic capture
   ↓
6. Timeout: Sets 30s timeout
   ↓
7. CORS: Validates origin and adds headers
   ↓
8. RateLimiter: Checks request limit
   ↓
9. SecureHeaders: Adds security headers
   ↓
10. APIVersion: Adds version header
   ↓
11. ContentTypeJSON: Validates and sets Content-Type
   ↓
12. Handler: Processes request (your code)
   ↓
13. Logger: Logs request end (status, latency)
   ↓
14. Response to client
```

---

## ⚠️ Important Order

The order of middlewares is critical:

1. **RealIP first**: Other middlewares need the real IP
2. **RequestID second**: Logger needs the ID for correlation
3. **Logger after RequestID**: To include ID in logs
4. **Recoverer after Logger**: To be able to log panics
5. **RateLimiter after RealIP**: To use the correct IP
6. **ContentTypeJSON at the end**: To validate before handler

---

## 🧪 Complete Request Example

**Request**:
```http
POST /api/v1/orders HTTP/1.1
Host: localhost:8080
Content-Type: application/json
X-Forwarded-For: 203.0.113.1
Origin: https://example.com

{"customer_id": "123", "items": [...]}
```

**Processing**:
1. ✅ RealIP: IP = 203.0.113.1
2. ✅ RequestID: ID = "550e8400-..."
3. ✅ Logger: Log start
4. ✅ Recoverer: Protection active
5. ✅ Timeout: 30s timer started
6. ✅ CORS: Origin allowed
7. ✅ RateLimiter: Client within limit
8. ✅ SecureHeaders: Headers added
9. ✅ APIVersion: Header added
10. ✅ ContentTypeJSON: Content-Type valid
11. ✅ Handler: Processes request
12. ✅ Logger: Log end (201 Created, 45ms)

**Response**:
```http
HTTP/1.1 201 Created
X-Request-ID: 550e8400-e29b-41d4-a716-446655440000
X-API-Version: 1.0.0
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Content-Type: application/json
Access-Control-Allow-Origin: https://example.com

{"success": true, "data": {...}}
```

---

## 📊 Middleware Summary

| # | Middleware | Purpose | Executes Before/After |
|---|------------|---------|----------------------|
| 1 | RealIP | Extracts real IP | Before |
| 2 | RequestID | Generates unique ID | Before |
| 3 | Logger | Logs request | Before and After |
| 4 | Recoverer | Captures panics | During |
| 5 | Timeout | Limits time | During |
| 6 | CORS | Handles CORS | Before |
| 7 | RateLimiter | Limits requests | Before |
| 8 | SecureHeaders | Security headers | Before |
| 9 | APIVersion | API version | Before |
| 10 | ContentTypeJSON | Validates Content-Type | Before |

---

## 🔧 Customization

You can adjust the configuration of some middlewares:

**RateLimiter**:
```go
config := middleware.RateLimiterConfig{
    RequestsPerSecond: 20,  // More permissive
    Burst: 50,
    KeyFunc: func(r *http.Request) string {
        // Rate limit by user instead of IP
        return getUserID(r)
    },
}
r.Use(middleware.RateLimiter(config))
```

**Timeout**:
```go
r.Use(chimiddleware.Timeout(60 * time.Second))  // More time
```

**CORS**:
```go
r.Use(cors.Handler(cors.Options{
    AllowedOrigins: []string{"https://myapp.com"},
    // ... more options
}))
```
