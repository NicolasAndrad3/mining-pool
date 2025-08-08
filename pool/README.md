# Mining Pool Backend

A production-grade backend for a mining pool, built with a modular architecture using **Go**, **Python**, and **Solidity**.  
The system is designed for real blockchain mining scenarios, with PostgreSQL storage, Prometheus metrics, share validation, and payment processing via smart contracts.

This repository is organized for clarity, extensibility, and maintainability, keeping each responsibility in its own module.

## Features

- **Go backend** as the main orchestrator.
- **Python services** for fraud detection and advanced share validation.
- **Solidity smart contract** interface for on-chain payments.
- PostgreSQL storage for shares and pool data.
- Prometheus metrics endpoint.
- Structured logging with text or JSON output.
- Configurable via `.env` or environment variables.
- Graceful shutdown handling.
- Middleware chain for request ID, structured logging, CORS, token authentication, security headers, timeouts, and request logging.
- Ready for adding advanced modules such as anti-fraud, off-chain processing, and blockchain integration.

## Architecture

pool/
├── core/ # Pool logic and share processing (Go)
├── http/ # HTTP server, routing, middlewares (Go)
├── database/ # PostgreSQL connection and storage (Go)
├── logs/ # Structured logger (Go)
├── security/ # Authentication and secrets (Go)
├── smartcontract/ # Smart contract payment interface (Go + Solidity)
├── metrics/ # Prometheus metrics registry (Go)
├── python/ # Fraud detection & validation service (Python)
├── config/ # Configuration loader (Go)
└── main.go # Main entry point (Go)


## Technology Stack

- **Go** — main backend engine
- **Python** — fraud detection and share validation services
- **Solidity** — payment processing via smart contracts
- **PostgreSQL** — persistent share and pool storage
- **Prometheus** — metrics collection

## Requirements

- Go 1.22+
- Python 3.10+
- Node.js (for Solidity compilation & deployment if needed)
- PostgreSQL
- Git

## Installation

1. Clone the repository:

       git clone https://github.com/NicolasAndrad3/mining-pool.git
       cd mining-pool

2. Create a .env file:

       SERVER_HOST=0.0.0.0
       SERVER_PORT=8080
       DATABASE_URL=postgres://mineruser:minerpass123@localhost:5432/miningpool?sslmode=disable
       API_KEY=testapikey_123456
       AUTH_TOKEN=default-token

3. Install Go dependencies:

       go mod tidy

4. Install Python dependencies:

   Running

Start the Go project:

     ./pool

Run the Python validation service:

     cd python
     python runner.py

If the Solidity contract is deployed, the backend will connect to it for processing payouts.

Endpoints:

/health
Check service health.

/submit
Accepts mining shares from workers.
Method: POST
Auth: Bearer Token in Authorization header.

/stats
Returns pool statistics.

/metrics
Prometheus metrics endpoint.

/test-payout
Simulates a payout using the payment engine.

Startup Flow
Load configuration.

Initialize logger.

Connect to PostgreSQL.

Load security secrets.

Initialize payment engine (Solidity-based or stub).

Start core pool logic.

Launch HTTP server with middleware stack.

Start Python validation service (fraud detection, share verification).

Serve metrics for Prometheus.

Production Roadmap
Implement production-ready Solidity payment engine.

Add full anti-fraud detection pipeline in Python.

Automatic database migrations.

Public key/signature-based authentication.

Full integration and load testing.

CI/CD pipeline setup.

TLS configuration and load balancing.

License
MIT License. Free to use and modify.
