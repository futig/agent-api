# Agent Backend

**Interview-to-Business-Requirements Agent** - Backend service that orchestrates LLM-based interview workflow to convert stakeholder conversations into standardized business requirements.

## Overview

This service provides two workflow modes:
- **Interview Mode**: Structured Q&A session with iterative question generation and validation
- **Draft Mode**: Free-form message collection with intelligent analysis and clarification

The system integrates with external LLM, RAG, and ASR services to:
- Generate contextual questions based on project documentation
- Validate completeness of collected information
- Generate structured business requirements in multiple formats (Markdown, JSON, DOCX, PDF)
- Support voice input through audio transcription
- Store and index requirements for future retrieval

## Tech Stack

### Backend
- **Go 1.25.1** - Primary language
- **Chi Router** - HTTP routing and middleware
- **PostgreSQL 16** - Primary database
- **pgx/v5** - PostgreSQL driver
- **sqlc** - Type-safe SQL code generation

### Architecture
- **Clean Architecture** - 4 layers (API → Usecase → Repository → Database)
- **Dependency Injection** - Builder pattern in `/internal/builder`
- **Interface at Point of Use** - Interfaces defined in usecase packages

### External Services
- **LLM Service** - Question generation, validation, summary creation
- **RAG Service** - Document indexing and context retrieval
- **ASR Service** - Audio transcription (WAV format)
- **Callback Service** - Async operation notifications

### Telegram Integration
- **go-telegram-bot-api/v5** - Bot framework
- **State Machine** - 15+ session states
- **7 Handlers** - Goal, Questions, Draft, Context, Project Save, Callback
- **Middleware** - Rate limiting, logging, recovery

## Quick Start

### Prerequisites

- Go 1.25.1+
- PostgreSQL 16+
- (Optional) Docker & Docker Compose

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd agent-backend
```

2. Copy environment configuration:
```bash
cp .env.example .env.local
```

3. Configure environment variables in `.env.local`:
   - Set `DATABASE_URL` to your PostgreSQL instance
   - Set `TELEGRAM_BOT_TOKEN` if using Telegram bot
   - Configure external service URLs (or use `ENABLE_MOCKS=true` for development)

4. Start PostgreSQL (if using Docker):
```bash
docker-compose up -d postgres
```

5. Run migrations (automatic on startup) and start the server:
```bash
make run-local
```

The HTTP API will be available at `http://localhost:8080`.

### Using Mock Services

For development without external services, set in `.env.local`:
```bash
ENABLE_MOCKS=true
```

This enables mock implementations of LLM, RAG, and ASR services.

#### Bot Features
- **Two workflow modes**: Interview and Draft
- **Voice support**: Send voice messages for answers
- **Project management**: Create and link sessions to projects
- **RAG integration**: Automatically indexes project files
- **Multi-format export**: Download as .md, .pdf
- **Skip questions**: Answer later if needed
- **Inline keyboards**: Button-based navigation

## Project Structure

```
agent-backend/
├── cmd/
│   ├── agent-backend/          # HTTP API server entrypoint
│   └── telegram-bot/           # Telegram bot entrypoint
├── internal/
│   ├── api/                    # HTTP handlers, routes, middleware
│   ├── builder/                # Dependency injection (builder pattern)
│   ├── config/                 # Configuration loading
│   ├── entity/                 # Domain models, DTOs, interfaces
│   ├── integration/            # External service connectors
│   │   ├── asr/                # Audio transcription service
│   │   ├── callback/           # Callback notification service
│   │   ├── llm/                # LLM service (questions, validation)
│   │   └── rag/                # RAG service (indexing, context)
│   ├── pkg/                    # Shared utilities
│   │   ├── formatter/          # Result formatters (MD, JSON, DOCX, PDF)
│   │   ├── http/               # HTTP client utilities
│   │   └── validator/          # Input validation
│   ├── repository/             # Database layer
│   │   ├── migrations/         # SQL migrations
│   │   ├── queries/            # sqlc query definitions
│   │   └── sqlc/               # Generated code (~1200 LOC)
│   ├── telegram/               # Telegram bot implementation
│   │   ├── bot/                # Core bot logic
│   │   ├── handlers/           # 7 state-specific handlers
│   │   ├── keyboard/           # Inline keyboard builder
│   │   ├── middleware/         # Rate limiting, logging, recovery
│   │   ├── render/             # Message templates
│   │   └── state/              # Session state management
│   └── usecase/                # Business logic
│       ├── project/            # Project management
│       └── session/            # Session orchestration
├── .env.example                # Environment variables template
├── .env.local                  # Development configuration
├── .env.prod                   # Production configuration
├── docker-compose.yml          # PostgreSQL + app services
└── sqlc.yaml                   # SQL code generation config
```

## Session Flow

### Interview Mode
```
NEW → ASK_USER_GOAL → SELECT_OR_CREATE_PROJECT →
ASK_USER_CONTEXT → GENERATING_QUESTIONS →
WAITING_FOR_ANSWERS → VALIDATING →
[loop if incomplete] → GENERATING_REQUIREMENTS → DONE
```

### Draft Mode
```
NEW → ASK_USER_GOAL → SELECT_OR_CREATE_PROJECT →
DRAFT_COLLECTING → [collect messages] →
VALIDATING → [generate questions if needed] →
WAITING_FOR_ANSWERS → GENERATING_REQUIREMENTS → DONE
```