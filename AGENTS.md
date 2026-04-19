# BethAPI - Agent Customization Guide

**BethAPI** is an AI Video Orchestration Engine—a stateful text-to-video generation backend built with Go and Google Agent Development Kit (ADK). It automates creative workflows through agentic orchestration, credit tracking, and tiered subscriptions.

## 🏗️ Project Architecture

### Layered Structure
- **Handlers** (API) → **Services** (Business logic) → **Repositories** (Data) → **Database** (Persistence)
- **Workers** (Asynq) → **Services** → **Repositories** for async job processing

### Key Components

| Component | Purpose | Location |
|-----------|---------|----------|
| **VideoAgent** | Orchestrates video generation pipeline via Google ADK | `agents/video_agent.go` |
| **Credit System** | Atomic credit deduction with transaction logging | `billing/credits.go` |
| **Subscriptions** | Tiered Pro/Ultra plans with auto-renewal + 48hr grace | `billing/subscription.go` |
| **Background Jobs** | High-reliability async processing via Asynq+Redis | `worker/` |
| **Auth** | Email/Password, Google OAuth, OTP, API Key support | `api/services/auth_service.go` |
| **Storage** | Cloudflare R2 (S3-compatible) with presigned URLs | `api/services/storage.go` |
| **Email** | OTP, payment notifications, grace period alerts | `api/services/email_service.go` |

## 🚀 Build & Run

### Prerequisites
- Go 1.25.5+
- MongoDB (local or Atlas)
- Redis (local or Cloud)
- Google AI Studio API Key
- Environment variables configured (see `.env.example`)

### Commands
```bash
# Install dependencies
go mod tidy

# Run server (localhost:8080)
go run main.go

# Run with custom port
PORT=3000 go run main.go
```

## 🤖 Agent Workflow (The Core)

### Pipeline
1. **User** → `POST /generate` with prompt
2. **Handler** → Validates, enqueues Asynq job, returns job ID
3. **Worker** (`video_worker.go`) → Dequeues, starts VideoAgent
4. **VideoAgent** → Uses Google ADK with Gemini 2.5 Flash
5. **Tools** → `PromptEnhancerTool` (refine), `UsageTrackerTool` (track credits)
6. **Future** → Veo 3.0 (video) + Imagen 3 (storyboard) integration

### Real-time Progress
- Client connects: `GET /jobs/{id}/stream` (Server-Sent Events)
- Server publishes updates via Redis pub/sub
- Stream closes when job completes

### Current Status
⚠️ **Proof-of-concept**: Agent orchestration and credit tracking working. Actual video generation (Veo/Imagen integration) not yet implemented.

## 📋 Key Files to Know

| File | Purpose |
|------|---------|
| `main.go` | Entry point; initializes services, database, server, worker |
| `config/config.go` | Viper-based config loading (33+ env vars) |
| `agents/video_agent.go` | VideoAgent definition and tool registration |
| `agents/tools/*.go` | Agent tools (PromptEnhancerTool, UsageTrackerTool) |
| `api/handlers/generate_handler.go` | Video generation endpoint |
| `api/handlers/sse_handler.go` | Real-time progress streaming |
| `api/services/auth_service.go` | User signup, login, JWT generation |
| `billing/credits.go` | Credit addition/deduction logic |
| `billing/subscription.go` | Subscription renewal and grace period handling |
| `worker/video_worker.go` | Asynq job processing |
| `api/models/user.go` | User schema and credit tracking |
| `api/repository/transaction_repository.go` | Audit trail for credit movements |

## 🔐 Authentication & Authorization

### Supported Auth Methods
1. **Email/Password**: Signup (50 free credits), Login with JWT
2. **Google OAuth**: Delegated auth
3. **OTP**: Via Resend email service
4. **API Key**: For programmatic access

### Auth Middleware
- JWT validation on protected endpoints
- Credit check before video generation
- Automatic credit deduction on job completion

### User Model
```go
type User struct {
    ID            string  // MongoDB ObjectID
    Email         string  // Unique
    PasswordHash  string  // bcrypt
    Credits       float64 // Atomic deductions
    SubscriptionID string // Tiered plan
    Verified      bool    // Email verification
}
```

## 💳 Billing System

### Credit Model
- **Signup**: 50 free credits
- **Deduction**: Per-job cost in tokens (tracked via UsageTrackerTool)
- **Transactions**: Immutable audit log

### Subscriptions
- **Pro** ($23/month): Monthly renewal
- **Ultra** ($290/month): Higher tier
- **Grace Period**: 48 hours for failed payments
- **Auto-renewal**: Background job every midnight

### Services
- **CreditService**: `AddCredits()`, `DeductCredits()`, `GetBalance()`
- **SubscriptionService**: `ProcessRenewal()`, `CheckGracePeriod()`, `EmailWarnings()`

## 📦 Data Models

### Databases
- **MongoDB**: Users, Jobs, Transactions collections
- **Redis**: Asynq task queue, agent state, pub/sub channels

### Key Collections
```
users {id, email, passwordHash, credits, subscriptionId, verified}
jobs {id, userId, prompt, status, progress, createdAt, completedAt}
transactions {id, userId, amount, type, description, createdAt}
```

## 🎯 Common Development Tasks

### Add a New Endpoint
1. Define DTO in `api/dto/`
2. Create handler in `api/handlers/`
3. Add service method in `api/services/`
4. Register route in `main.go`
5. Add auth middleware if needed

### Add a New Agent Tool
1. Create tool struct in `agents/tools/`
2. Implement `tool.Execute()` method
3. Register in `VideoAgent.RegisterTools()` in `agents/video_agent.go`
4. Test with `POST /generate`

### Modify Credit System
1. Update cost in `UsageTrackerTool.Execute()`
2. Test deduction via `billing/credits.go`
3. Verify transaction logging in MongoDB

### Add a New Service
1. Create service file in `api/services/`
2. Inject into handler constructor
3. Initialize in `main.go` before server startup

## ⚡ Important Patterns

### Dependency Injection
All services use constructor injection. Example:
```go
type GenerateHandler struct {
    videoService *services.VideoService
    creditService *services.CreditService
}

func NewGenerateHandler(vs *services.VideoService, cs *services.CreditService) *GenerateHandler {
    return &GenerateHandler{videoService: vs, creditService: cs}
}
```

### DTO Validation
All incoming requests use `go-playground/validator`:
```go
type GenerateRequest struct {
    Prompt string `json:"prompt" validate:"required,min=10"`
}
```

### Repository Pattern
Data access abstracted via repositories:
```go
type UserRepository interface {
    FindByEmail(email string) (*User, error)
    Save(user *User) error
}
```

### Error Handling
Structured error responses with HTTP status codes:
```go
c.JSON(400, map[string]string{"error": "Invalid prompt"})
```

## ⚠️ Pitfalls & Considerations

1. **Google ADK State**: Agent state is ephemeral. For persistence, ensure tool responses are captured.
2. **Redis Connection**: Jobs depend on Redis connectivity. No fallback queue implemented.
3. **Video Generation TBD**: Veo 3.0 and Imagen 3 integration not yet implemented—placeholder only.
4. **Credit Precision**: Use `float64` for credits; rounding may occur in edge cases.
5. **Subscription Renewal**: Runs at midnight UTC. Ensure server timezone is UTC.
6. **Presigned URLs**: R2 URLs expire in 24 hours. Don't cache longer.
7. **SSE Connections**: Browser limits 6 concurrent connections per domain. Consider connection pooling.

## 🔧 Environment Variables

Essential variables (see `.env.example` for full list):
```
# Database
MONGODB_URI=mongodb://localhost:27017
REDIS_URL=redis://localhost:6379

# Google AI
GOOGLE_API_KEY=your_api_key

# Storage
R2_BUCKET_NAME=bethapi
R2_ACCOUNT_ID=your_account_id
R2_ACCESS_KEY_ID=your_key
R2_SECRET_ACCESS_KEY=your_secret

# Email
RESEND_API_KEY=your_key

# Auth
JWT_SECRET=your_secret

# Payments
FLUTTERWAVE_SECRET_KEY=your_key
PAYSTACK_SECRET_KEY=your_key

# Server
PORT=8080
```

## 📖 Related Documentation

- [README.md](README.md) – Project overview, features, getting started
- `PROJECT_ANALYSIS.md` – Detailed technical analysis (if present)

---

**Last Updated**: April 2026  
**For**: AI coding agents working on video generation features, billing logic, or agent orchestration
