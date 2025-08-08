# Mining Pool Backend

![Banner](https://github.com/user-attachments/assets/623312c9-b87f-4a48-a3b4-5f3354989bf5)

## Overview

High-performance mining pool backend using **Go** and **Python**, integrated with PostgreSQL and Docker. Designed for real mining scenarios with share validation, fraud detection, and optional smart contract payments.

## Features

* **Go** for core mining logic and API
* **Python** for fraud detection & validation service
* PostgreSQL storage
* Docker support for deployment
* Modular, production-ready architecture

## Structure

```
pool/
 ├── cmd/            # Main entry point (Go)
 ├── core/           # Mining logic (Go)
 ├── database/       # PostgreSQL integration (Go)
 ├── http/           # HTTP API server (Go)
 ├── logs/           # Logging system (Go)
 ├── security/       # Security & antifraud (Go)
 ├── python/         # Fraud detection & share validation (Python)
 └── Dockerfile      # Docker build configuration
```

## Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/NicolasAndrad3/mining-pool.git
cd mining-pool
```

### 2. Environment variables

Create a `.env` file:

```env
DATABASE_URL=postgres://user:password@localhost:5432/miningpool?sslmode=disable
API_KEY=your_api_key
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
```

### 3. Run locally

```bash
go build -o pool ./cmd/main.go
./pool
```

### 4. Run with Docker

```bash
docker build -t mining-pool .
docker run -p 8080:8080 --env-file .env mining-pool
```

## License

MIT License
