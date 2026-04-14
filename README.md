# BethAPI - AI Video Orchestration Engine

BethAPI is a robust, stateful text-to-video generation backend built with Go and the Google Agent Development Kit (ADK). It orchestrates complex creative workflows by leveraging Gemini 2.5 Flash, Veo 3.0, and Imagen 3.

## 🚀 Features

- **Agentic Orchestration**: Stateful agents using Google ADK to plan and execute multi-step video generation (Prompt Enhance -> Storyboard -> Gen -> Stitch).
- **Real-time Credit Tracking**: Atomic credit deduction system synchronized with AI Studio usage metadata.
- **Tiered Subscriptions**: Monthly Pro ($23) and Ultra ($290) plans with automated renewal and a 48-hour grace period.
- **Async Processing**: High-reliability background jobs using Asynq (Redis) with real-time progress via SSE.
- **Global Storage**: Seamless integration with Cloudflare R2 for secure video hosting.
- **Auth Systems**: Support for Email/Password, Google OAuth, OTP (Resend), and API Key authentication.

## 🛠 Tech Stack

- **Backend**: Go + Echo
- **Agentic Layer**: Google Agent SDK (ADK) 1.1.0
- **AI Models**: Google AI Studio (Gemini 2.5 Flash, Veo 3.0, Imagen 3)
- **Database**: MongoDB
- **Queue/Worker**: Redis + Asynq
- **Storage**: Cloudflare R2 (S3-compatible)
- **Payments**: Flutterwave & Paystack
- **Email**: Resend

## 🏁 Getting Started

### Prerequisites

- Go 1.25.5 or higher
- MongoDB and Redis instances
- Google AI Studio API Key
- Cloudflare R2 Credentials
- Resend API Key

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/your-repo/bethapi.git
   cd bethapi
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Setup environment variables:
   ```bash
   cp .env.example .env
   # Edit .env with your credentials
   ```

4. Run the server:
   ```bash
   go run main.go
   ```

## 📡 API Endpoints

### Authentication
- `POST /api/v1/signup` - Register a new user (receives 50 free credits)
- `POST /api/v1/login` - Standard email/password login
- `GET /api/v1/me` - Get current user profile and credit balance

### Video Generation
- `POST /api/v1/generate` - Enqueue a new video generation job
- `GET /api/v1/jobs/:id/stream` - SSE stream for real-time orchestration progress

### Billing
- `POST /api/v1/topup` - Generate payment link for one-time credits
- `POST /api/v1/subscribe` - Initiate monthly Pro/Ultra subscription

## 🧠 Agentic Architecture

The system uses **Google ADK** to manage stateful tool-use. When a job is enqueued:
1. An **Orchestrator Agent** is initialized with the User's context.
2. It sequentially calls the `prompt_enhancer` (Gemini) and the generation tools.
3. After each tool execution, the `usage_tracker` tool deducts credits based on actual token/second consumption.
4. Agent state is persisted in MongoDB, allowing for long-running job monitoring and recovery.

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.
