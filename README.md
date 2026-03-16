# Stock Analyzer Pro 🚀

A professional-grade stock analysis platform that provides real-time stock ratings, dynamic price charts, and advanced investment recommendations using an automated scoring engine.

## ✨ Features

- **Advanced Recommendation Engine**: A custom algorithm that scores stocks based on:
  - **Upside Potential**: Live calculation of current price vs. analyst targets.
  - **Time Decay**: Recent ratings are weighted more heavily than stale ones.
  - **Rating Momentum**: Bonuses for recent upgrades.
- **Real-Time Data**: Integrated with **Alpaca Market Data API** for live prices and news.
- **Modern Dashboard**: A responsive Vue.js 3 frontend with centered mini-charts and deduplicated upgrade feeds.
- **Cloud-Native Architecture**: Fully serverless deployment on **AWS Lambda** and **API Gateway**.
- **Resilient Storage**: Powered by **CockroachDB Serverless** with secure SSL/TLS connectivity.
- **Cost Optimized**: Zero-cost architecture using AWS Free Tier and automated budget monitoring ($0.01 alert).

## 🛠 Tech Stack

- **Backend**: Go 1.23+, Gin Gonic, SQL (PostgreSQL & SQLite compatible)
- **Frontend**: Vue.js 3, TypeScript, TailwindCSS, Pinia
- **Infrastructure**: Terraform, AWS Lambda, S3, CloudFront
- **Database**: CockroachDB (Production), SQLite (Local)

## 🚀 Quick Start

### Local Development
1. Clone the repository.
2. Setup your `.env` in the `backend/` folder with Alpaca credentials.
3. Run the backend: `cd backend && go run cmd/lambda/main.go`
4. Run the frontend: `cd frontend && npm install && npm run dev`

### Deployment
Infrastructure is managed via Terraform in `backend/terraform`. Deployments are automated via scripts in `backend/scripts/deploy.sh`.

## 🔒 Security & Performance
- **SSL/TLS**: All database connections are encrypted using bundled root certificates.
- **CORS**: Restricted to specific production domains.
- **Caching**: 5-minute intelligent caching for recommendation results to stay under API limits.

---
**AI-Generated investment recommendations based on analyst consensus and market data.**
