# Stock Analyzer Pro 🚀

A professional-grade real-time investment intelligence platform. This system automatically ingests analyst sentiment and market data to provide a dynamic "Investment Score" for the world's most traded stocks.

<img width="2786" height="1700" alt="image" src="https://github.com/user-attachments/assets/fe433757-26d7-4c51-abc2-4960aea723b4" />

## 🧠 Advanced Recommendation Engine
Unlike basic trackers, this platform uses a custom **Weighted Scoring Algorithm** to identify opportunities:
- **Real-Time Upside Analysis**: Calculates the exact percentage gap between current market prices and analyst targets.
- **Time-Decay Weighting**: Ratings follow an exponential decay curve; sentiment from "Issued Today" is prioritized over stale data.
- **Action Momentum**: Stocks receiving recent "Upgrades" get a momentum bonus in their overall score.
- **AI-Driven Rationale**: Every recommendation includes a human-readable explanation of its score (e.g., *"Upgraded to Buy with a 12.4% upside potential"*).

## ✨ Key Functionalities
- **Live Market Feeds**: Integration with **Alpaca Market Data** for real-time prices and news sentiment.
- **Smart Dashboard**:
  - **Mini-Charts**: High-density SVG sparklines showing the last 7 days of price action.
  - **Deduplicated Feeds**: Intelligently shows only the latest relevant rating per company.
- **Automated Ingestion**: A serverless "Keep-Alive" scheduler that fetches new data every 4 hours and ensures the cloud database stays warm.

## 🛠 Cloud Architecture (Free-Tier Optimized)
The project is built to run **100% free** while maintaining professional standards:
- **Compute**: AWS Lambda (Go 1.23+) triggered via API Gateway.
- **Storage**: CockroachDB Serverless (Cloud Postgres) with SSL/TLS encryption.
- **Infrastructure**: Fully managed via Terraform (IaC).
- **Financial Armor**:
  - **Hard Throttling**: API Gateway limits prevent expensive traffic spikes.
  - **Zero-Tolerance Budget**: AWS Budget monitoring with a **$0.01 alert threshold**.
  - **Concurrency Limits**: Restricted Lambda scaling to prevent billing surprises.

## 🚀 Quick Start

### Local Development
1. Clone the repo and install dependencies (`npm install` in frontend).
2. Configure `.env` with your Alpaca API Keys and Database URL.
3. Run local backend: `cd backend && go run cmd/lambda/main.go`
4. Run local frontend: `cd frontend && npm run dev`

### Production Deployment
Automatic deployment scripts are located in `backend/scripts/deploy.sh`.

---
**Disclaimer**: AI-Generated investment recommendations based on analyst consensus and market data. For educational purposes only.
