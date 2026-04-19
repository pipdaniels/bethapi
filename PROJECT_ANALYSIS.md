# BethAPI Project Analysis

## Overview
BethAPI is a **stateful AI video orchestration engine** built in Go that uses Google Agent Development Kit (ADK) to manage complex multi-step video generation workflows. It combines LLM intelligence with video synthesis capabilities through Gemini 2.5 Flash, Veo 3.0, and Imagen 3.

---

## 1. MAIN ENTRY POINT & ARCHITECTURE

### Entry Point: `main.go`
**Initialization Sequence:**
1. **Load Configuration** → Reads `.env` or environment variables via godotenv
2. **Initialize Databases**:
   - MongoDB connection for persistent data (users, jobs, transactions)
   - Asynq (Redis) setup for async job queue
   - R2 (Cloudflare) storage service initialization
   - Email service (Resend)
3. **Initialize Services & Repositories**:
   - `UserRepository` for database operations
   - `AuthService` for signup/login/JWT management
4. **Create Google GenAI Client** → Connects to Gemini API backend
5. **Start Background Worker** → Spawns async task processor (Asynq server with 10 concurrency)
6. **Set up Echo Web Server** with middleware:
   - Logging, Recovery, CORS
   - Custom auth middleware (JWT + API Key support)
   - Credit check middleware for generation endpoints
7. **Graceful Shutdown** → Handles OS signals for clean shutdown

### Architecture Pattern
- **Layered Architecture**: Handlers → Services → Repositories → Database
- **Async-First Design**: All video generation jobs queued via Asynq/Redis
- **Agent-Centric**: VideoAgent orchestrates multi-step workflows using ADK
- **Dependency Injection**: Services receive repositories/clients in constructors

---

## 2. API ENDPOINT STRUCTURE & ROUTING

### Base Route: `/api/v1`

#### Authentication Endpoints
| Endpoint | Method | Auth Required | Purpose |
|----------|--------|---------------|---------|
| `/signup` | POST | ❌ | Register user (receives 50 free credits) |
| `/login` | POST | ❌ | Email/password authentication → JWT token |
| `/me` | GET | ✅ JWT | Fetch current user profile & credit balance |

**Auth Middleware Flow:**
1. Extract `Authorization: Bearer <token>` header
2. Validate JWT signature using `JWT_SECRET`
3. Extract email from claims
4. Load user from MongoDB (`users` collection)
5. Attach user object to Echo context (`c.Get("user")`)

#### Video Generation Endpoints
| Endpoint | Method | Auth | Middleware | Purpose |
|----------|--------|------|-----------|---------|
| `/generate` | POST | ✅ | JWT + CreditCheck | Enqueue video generation job |
| `/jobs/:id/stream` | GET | ❌ | - | Real-time progress via Server-Sent Events (SSE) |

**Generate Handler Flow:**
1. Parse `GenerateRequest` (prompt, duration, aspect_ratio, reference_image_url)
2. Extract user from context
3. Generate unique `jobID` (MongoDB ObjectID)
4. **Create job record in MongoDB** with status `pending` via `JobRepository.Create()`
5. Enqueue `VideoGenerationPayload` to Asynq with type `"video:generate"`
6. If enqueue fails, immediately mark job as `failed` in MongoDB
7. Return `202 Accepted` with jobID

**JobStream Handler (SSE):**
- Sets SSE headers
- Fetches current job record from MongoDB immediately on connect
- If job is already `completed` or `failed`, sends final state and closes
- **Subscribes to Redis Pub/Sub channel** `job:{jobID}` for real-time push
- Streams each published event the instant the worker posts it (zero polling lag)
- Unsubscribes and closes when the stream reaches a terminal state or the client disconnects

#### Request/Response DTOs
```go
// Generate Request
{
  "prompt": "string",           // Required
  "duration": 0.0,              // Optional
  "aspect_ratio": "16:9",       // Optional
  "reference_image_url": "url"  // Optional
}

// Generate Response
{ "job_id": "uuid-string" }

// Job Status (via SSE)
{
  "job_id": "uuid",
  "status": "processing|completed|failed",
  "progress": 0.0,
  "video_url": "r2-url",        // When complete
  "credits_used": 10.5
}
```

---

## 3. AGENT WORKFLOW & VIDEO GENERATION PIPELINE

### VideoAgent Orchestration (`agents/video_agent.go`)

**Architecture:**
- Uses Google ADK (`google.golang.org/adk/agent/llmagent`)
- Model: Gemini 2.0 Flash
- Name: `VideoOrchestrator`
- Status: **Proof-of-concept** (minimal implementation)

**Agent Tools:**
1. **PromptEnhancerTool** - Gemini-powered prompt enhancement
2. **UsageTrackerTool** - Token counting & credit deduction

**Agent Instruction:**
```
"Follow these steps: 
1. Enhance the prompt using prompt_enhancer. 
2. Track usage."
```

### Video Generation Pipeline (Conceptual Flow)

```
User Input (POST /generate)
    ↓
[GenerateHandler]
    ↓
Enqueue Asynq Task: TypeVideoGeneration
    ↓
[VideoWorker.ProcessTask]
    ↓
[VideoAgent.Run]
    ├─ PromptEnhancerTool (Gemini 2.0 Flash)
    │   └─ Input: Simple prompt
    │   └─ Output: Detailed scene description (cinematic details, lighting, camera movement)
    │
    ├─ UsageTrackerTool
    │   └─ Input: UsageMetadata (prompt_tokens, output_tokens, video_duration_seconds)
    │   └─ Calculates: Cost = (prompt_tokens/1000 × PricingLLMPrompt1K) + 
    │                         (output_tokens/1000 × PricingLLMOutput1K) +
    │                         (duration_sec × PricingVideoSec)
    │   └─ Deducts credits from user
    │   └─ Output: Confirmation + cost breakdown
    │
    └─ [Future] Veo 3.0 Generation Tool
    └─ [Future] Imagen 3 Generation Tool
    └─ [Future] Video Stitching Tool

Agent State (Redis via ADK):
    ├─ user_id
    ├─ job_id
    ├─ prompt
    ├─ enhanced_prompt
    ├─ credits_deducted
    └─ video_url
```

### Tool Implementations

#### PromptEnhancerTool
```go
Input: "Make a car driving"
System: "Act as professional cinematographer. Rewrite into detailed scene description..."
Output: "A sleek black sedan accelerates down a winding mountain road at golden hour. 
         Camera mounted on hood showing glossy paint reflecting warm sunlight. 
         Dust particles suspended in headlight beams. Low angle emphasizing speed..."
```

#### UsageTrackerTool
```go
Input: UsageMetadata{
  PromptTokenCount: 150,
  CandidatesTokenCount: 2500,
  VideoDurationSeconds: 30.5
}
Calculation:
  promptCost = (150/1000) × PRICING_LLM_PROMPT_1K
  outputCost = (2500/1000) × PRICING_LLM_OUTPUT_1K
  videoCost = 30.5 × PRICING_VIDEO_SEC
  totalCost = promptCost + outputCost + videoCost
Output: "Deducted X.XX credits for user. Usage: 150 prompt tokens, 2500 output tokens, 30.5 video sec."
```

**Current Status:**
- ✅ Prompt enhancement working
- ✅ Usage tracking architecture in place
- ❌ Actual Veo/Imagen generation **not implemented** (framework only)
- ❌ Credit deduction not yet activated (commented in tool)
- ❌ Job status persistence in MongoDB commented out

---

## 4. KEY SERVICES

### AuthService (`api/services/auth_service.go`)
**Methods:**
- `Signup(ctx, SignupRequest)` → Creates user with bcrypt-hashed password + 50 free trial credits + API key
- `Login(ctx, LoginRequest)` → Validates credentials, returns JWT token
- `GenerateToken(user)` → Creates JWT with user email in claims

**JWT Configuration:**
- Secret: From `JWT_SECRET` env variable
- Claims: `email` (subject)
- Validity: Not explicitly set (review needed)

### CreditService (`billing/credits.go`)
**Methods:**
- `AddCredits(ctx, userID, amount, description)` → Increment balance + log transaction
- `DeductCredits(ctx, userID, amount, jobID, description)` → Decrement balance + log debit transaction

**Credit Types:**
- `TypeCredit` - Incoming (purchases, refunds)
- `TypeDebit` - Outgoing (usage)

### SubscriptionService (`billing/subscription.go`)
**Methods:**
- `ProcessRenewal(ctx, userID, plan)` → Reset balance to plan baseline, schedule next renewal (1 month)
  - Pro: 5000 credits/month
  - Ultra: 25000 credits/month
- `HandleFailedPayment(ctx, userID)` → Set status to `past_due`, record grace period start
- `DowngradeToFree(ctx, userID)` → Reset to free plan after failed renewal

**Subscription Models:**
```
PlanFree  = "free"     (50 init credits, no renewal)
PlanPro   = "pro"      (5000/month, $23)
PlanUltra = "ultra"    (25000/month, $290)

StatusActive   = "active"    (subscription valid)
StatusPastDue  = "past_due"  (payment failed, in grace period)
StatusCanceled = "canceled"  (user canceled)
```

**Billing Worker Logic** (`worker/billing_worker.go`):
- `CheckRenewals()` - Finds expiring subscriptions, triggers renewal (webhook-based)
- `HandleGracePeriods()` - Finds past_due users, sends reminder emails, downgrades after 48h grace period

### StorageService (`api/services/storage.go`)
**Purpose:** R2 (Cloudflare S3-compatible) file storage
**Methods:**
- `GetPresignedURL(ctx, key, expires)` → Generate temporary download links
- Upload method (partial implementation visible)

**Configuration:**
- Endpoint: `R2_ENDPOINT` (e.g., `https://xyz.r2.cloudflarestorage.com`)
- Bucket: `R2_BUCKET`
- Public Domain: `R2_PUBLIC_DOMAIN` (for public URLs)
- Region: "auto" (Cloudflare native)

### EmailService (`api/services/email_service.go`)
**Provider:** Resend (transactional email)
**Methods:**
- `SendOTP(to, code)` - 6-digit verification code
- `SendPaymentNotification(to, amount, credits)` - Payment receipt
- `SendGracePeriodWarning(to, hoursLeft, attemptNum)` - Subscription renewal warning

### JobPublisher (`api/services/job_publisher.go`)
**Purpose:** Redis Pub/Sub bridge between the background worker and SSE clients

**Functions:**
- `JobChannelName(jobID)` → channel key `job:{jobID}`
- `PublishJobUpdate(ctx, job)` → marshals job to JSON and publishes to its Redis channel
- `SubscribeJobUpdates(ctx, jobID)` → returns a `*redis.PubSub` the SSE handler listens on

**Real-time flow:**
```
Worker                        Redis              SSE Handler
  ├─ UpdateStatus (Mongo)       │                    │
  ├─ PublishJobUpdate ────────► job:{id} ──────────► <-sub.Channel()
                                                 ├─ Unmarshal
                                                 └─ SSE event → client
```

---

## 5. BUILD/RUN COMMANDS & SERVER STARTUP

### Prerequisites
```
Go 1.25.5
MongoDB instance
Redis instance
Google AI Studio API Key
Cloudflare R2 account
Resend API account
```

### Installation & Setup
```bash
# Clone repo
git clone <repo-url>
cd bethapi

# Install dependencies
go mod tidy

# Create .env file
cp .env.example .env
# Edit with credentials

# Run server
go run main.go
```

### Server Configuration
**Default Port:** `8080` (override via `PORT` env var)

**CORS Configuration:**
- Allowed origins from `ALLOWED_ORIGINS` env var (space-separated list)
- Allowed headers: Origin, Content-Type, Accept, Authorization, X-API-KEY

### Background Worker
**Asynq Server Configuration:**
- Redis connection: `REDIS_ADDR` (default: `localhost:6379`)
- Concurrency: 10 workers (processes 10 jobs in parallel)
- Task type: `"video:generate"`

**Graceful Shutdown:**
- Server listens for `SIGINT` (Ctrl+C)
- Logs shutdown message
- Closes all connections cleanly

---

## 6. DATABASE SETUP

### MongoDB
**Collections:**
- `users` - User accounts, subscriptions, credits
- `jobs` - Video generation jobs (fully tracked via `JobRepository`)
- `transactions` - Credit ledger
- (Implicit) Collections managed by Asynq via Redis

**User Model Schema:**
```go
{
  _id: ObjectID,
  email: string (unique),
  password: string (bcrypt hashed),
  name: string,
  credit_balance: float64,
  total_credits_used: float64,
  plan: "free" | "pro" | "ultra",
  subscription_status: "active" | "past_due" | "canceled",
  renews_at: Date (optional),
  grace_period_started: Date (optional),
  notification_count: int,
  api_key: string (unique),
  created_at: Date,
  updated_at: Date
}
```

**Transaction Model Schema:**
```go
{
  _id: ObjectID,
  user_id: ObjectID (foreign key),
  type: "credit" | "debit",
  amount: float64,
  tokens_used: int64 (optional),
  job_id: string (optional, foreign key),
  description: string,
  created_at: Date
}
```

### Redis
**Purpose:**
- **Asynq Task Queue** - Stores video generation jobs
- **Agent State** (potential) - Stores VideoAgent state during execution
- **Real-time Updates** - Pub/Sub for job progress

**Connection:**
```go
addr: config.AppConfig.RedisAddr  // Default: "localhost:6379"
```

**Asynq Inspector:**
- Monitor pending, active, completed tasks
- Retry failed jobs
- Inspect job payloads

### Connection Flow
```
main.go
  ├─ ConnectMongo() → MongoClient connected
  ├─ InitAsynq() → AsynqClient & Inspector created
  ├─ InitStorage() → R2 S3Client configured
  └─ InitEmail() → Resend client ready
```

---

## 7. IMPORTANT CONVENTIONS & PATTERNS

### Dependency Injection
All services receive dependencies via constructors:
```go
// ✅ Pattern used
authService := services.NewAuthService(userRepo)
genHandler := handlers.NewGenerateHandler(database.AsynqClient)

// Handlers receive services
authHandler := handlers.NewAuthHandler(authService)
```

### Error Handling
- **Echo handlers** return `dto.ErrorResponse` as JSON
- **Services/Repositories** return `error` type
- **No custom error types** (uses standard Go errors)
- Errors logged to stdout with `log.Printf()` / `log.Fatalf()`

### Repository Pattern
```go
type UserRepository struct {
  collection *mongo.Collection
}

// CRUD operations
func (r *UserRepository) Create(ctx, user) error
func (r *UserRepository) GetByEmail(ctx, email) (*User, error)
func (r *UserRepository) GetByID(ctx, id) (*User, error)
func (r *UserRepository) UpdateCredits(ctx, userID, amount) error
```

### Middleware Chaining
```go
// Multiple middleware applied to route groups
gen := v1.Group("/generate", 
  middleware.CombinedAuthMiddleware,
  middleware.CreditCheckMiddleware)
gen.POST("", genHandler.Generate)
```

### Data Transfer Objects (DTOs)
- Separate request/response types for API contracts
- Validation using `github.com/go-playground/validator/v10`
- Example:
  ```go
  type SignupRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Name     string `json:"name" validate:"required"`
  }
  ```

### Global Singletons
```go
// Database
database.MongoClient
database.AsynqClient
database.AsynqInspector

// Services
services.Storage    // R2
services.Email      // Resend

// Config
config.AppConfig    // Environment variables
```

### BSON Tags
- `bson:"field_name"` - MongoDB serialization
- `bson:"_id,omitempty"` - Primary key (omitted on creation)
- `json:"-"` - Password field not returned in API responses

---

## 8. CONFIGURATION & ENVIRONMENT SETUP

### Config Structure (`config/config.go`)
```go
type Config struct {
  // Server
  Port           string   // Default: "8080"
  AllowedOrigins []string // CORS origins
  
  // Databases
  MongoURI       string   // MongoDB connection string
  MongoDBName    string   // Database name
  RedisAddr      string   // Redis host:port
  
  // Authentication
  JWTSecret      string   // Secret for JWT signing
  
  // Third-party APIs
  GoogleAIKey    string   // Google AI Studio API key
  ResendAPIKey   string   // Resend email API key
  
  // Cloud Storage (Cloudflare R2)
  R2AccessKey    string
  R2SecretKey    string
  R2Endpoint     string   // Full URL to R2 endpoint
  R2Bucket       string
  R2PublicDomain string   // Base URL for public files
  
  // Pricing (Per-unit costs)
  PricingLLMPrompt1K  float64  // Cost per 1000 prompt tokens
  PricingLLMOutput1K  float64  // Cost per 1000 output tokens
  PricingVideoSec     float64  // Cost per second of video
  PricingImagen       float64  // Cost per Imagen generation
  
  // Subscriptions
  ProPriceID   string    // Payment processor price ID for Pro plan
  UltraPriceID string    // Payment processor price ID for Ultra plan
  
  // Billing Policy
  GracePeriodHours int   // Grace period after failed payment (e.g., 48)
  RenewalNotifyCount int // Number of reminder emails to send
}
```

### Loading Configuration
```go
func LoadConfig() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found, using environment variables")
	}

	AppConfig = &Config{
		// Manual mapping using GetEnv, GetEnvFloat, GetEnvInt helpers
	}
}
```

### Required Environment Variables
```bash
# Server
PORT=8080
ALLOWED_ORIGINS="http://localhost:3000 http://localhost:5173"

# MongoDB
MONGO_URI="mongodb+srv://user:pass@cluster.mongodb.net/db?retryWrites=true&w=majority"
MONGO_DB_NAME="bethapi"

# Redis
REDIS_ADDR="localhost:6379"

# Auth
JWT_SECRET="your-secret-key-min-32-chars"

# Google AI
GOOGLE_AI_API_KEY="your-gemini-api-key"

# Email (Resend)
RESEND_API_KEY="your-resend-key"

# Cloudflare R2
R2_ACCESS_KEY="your-r2-access-key"
R2_SECRET_KEY="your-r2-secret-key"
R2_ENDPOINT="https://xyz.r2.cloudflarestorage.com"
R2_BUCKET="bethapi"
R2_PUBLIC_DOMAIN="https://cdn.example.com"

# Pricing
PRICING_LLM_PROMPT_1K=0.00001
PRICING_LLM_OUTPUT_1K=0.00003
PRICING_VIDEO_SEC=0.1
PRICING_IMAGEN=0.05

# Subscriptions
SUBSCRIPTION_PRO_PRICE_ID="price_xyz"
SUBSCRIPTION_ULTRA_PRICE_ID="price_abc"

# Billing Policy
RENEWAL_GRACE_PERIOD_HOURS=48
RENEWAL_NOTIFY_COUNT=3
```

### Helper Functions
```go
GetEnv(key, fallback)       // string
GetEnvFloat(key, fallback)  // float64
GetEnvInt(key, fallback)    // int
```

---

## 9. ADVANCED FEATURES & FUTURE ROADMAP

### Implemented ✅
- JWT + API Key authentication
- Credit balance system
- Subscription plans (Pro, Ultra)
- Email notifications (Resend)
- Cloud storage (R2)
- Async job queue (Asynq)
- Agent framework (Google ADK)
- Prompt enhancement tool
- Usage tracking tool
- Job lifecycle tracking in MongoDB (pending → processing → completed/failed)
- JobRepository with Create, GetByID, UpdateStatus, MarkCompleted, MarkFailed, ListByUser
- Real-time SSE via Redis Pub/Sub — worker publishes on every transition; handler streams instantly
- **Billing worker (grace period logic)** — automated hourly checks for renewals and grace period notifications

### Partially Implemented 🟡
- Video URL and credit deduction in worker (populated when Veo is integrated)


## 10. CODE QUALITY OBSERVATIONS

### Strengths
✅ Clean separation of concerns (handlers, services, repositories)
✅ Interface-based dependency injection
✅ Consistent error handling patterns
✅ DTOs for API contracts
✅ MongoDB with proper context handling
✅ Graceful shutdown support

### Areas for Improvement
⚠️ Error messages could be more specific
⚠️ No request logging/tracing
⚠️ Global singletons could use interface patterns

---

## 11. QUICK START CHECKLIST

```
[ ] Go 1.25.5 installed
[ ] MongoDB running (local or cloud)
[ ] Redis running (local or cloud)
[ ] Create .env with all required variables
[ ] `go mod tidy` - install dependencies
[ ] `go run main.go` - start server
[ ] Test: POST /api/v1/signup → Create user
[ ] Test: POST /api/v1/login → Get JWT
[ ] Test: POST /api/v1/generate (with JWT) → Enqueue job
[ ] Monitor background worker for task processing
```

---

## Summary
BethAPI is a **well-architected video orchestration platform** with solid foundations in authentication, billing, and async processing. The Google ADK integration enables stateful agentic workflows, though the core video generation and synthesis features remain to be fully implemented. The codebase follows Go best practices and is ready for production with the addition of actual AI model integrations and payment processor webhooks.
