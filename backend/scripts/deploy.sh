#!/bin/bash

# Stock Analyzer Lambda Deployment Script
# This script builds and deploys the Go Lambda functions to AWS

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="stock-analyzer"
ENVIRONMENT="dev"
AWS_REGION="us-east-1"
S3_BUCKET="stock-analyzer-dev-lambda-deployments-lqtqgjg5"

# Directory paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BUILD_DIR="$PROJECT_ROOT/build"

main() {
    echo -e "${BLUE}🚀 Starting deployment for $PROJECT_NAME ($ENVIRONMENT)${NC}"

    # 1. Check prerequisites
    echo "🔍 Checking prerequisites..."
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Error: Go is not installed${NC}"
        exit 1
    fi
    if ! command -v aws &> /dev/null; then
        echo -e "${RED}Error: AWS CLI is not installed${NC}"
        exit 1
    fi
    if ! command -v zip &> /dev/null; then
        echo -e "${RED}Error: zip is not installed${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ All prerequisites satisfied${NC}"

    # 2. Build Go binary
    echo "🔨 Building Lambda function..."
    cd "$PROJECT_ROOT/cmd/lambda"
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap main.go
    
    # 3. Create deployment package
    echo "📦 Creating deployment package..."
    cd "$PROJECT_ROOT"
    mkdir -p build
    rm -f build/lambda-deployment.zip
    zip -j build/lambda-deployment.zip cmd/lambda/bootstrap
    if [ -f "certs/root.crt" ]; then
        zip -j build/lambda-deployment.zip certs/root.crt
    fi
    rm cmd/lambda/bootstrap
    echo -e "${GREEN}✓ Lambda function built successfully${NC}"

    # 4. Upload to S3
    echo "☁️  Uploading to S3..."
    aws s3 cp build/lambda-deployment.zip "s3://$S3_BUCKET/lambda-deployment.zip" --region "$AWS_REGION"
    echo -e "${GREEN}✓ Deployment package uploaded to S3${NC}"

    # 5. Update Lambda functions
    echo "🔄 Updating Lambda functions..."
    FUNCTIONS=("api" "ingestion" "scheduler")
    for func in "${FUNCTIONS[@]}"; do
        FUNCTION_NAME="$PROJECT_NAME-$ENVIRONMENT-$func"
        echo "Updating $FUNCTION_NAME..."
        aws lambda update-function-code \
            --function-name "$FUNCTION_NAME" \
            --s3-bucket "$S3_BUCKET" \
            --s3-key lambda-deployment.zip \
            --region "$AWS_REGION" \
            --no-cli-pager > /dev/null
    done
    echo -e "${GREEN}✓ Lambda functions updated successfully${NC}"

    echo -e "${GREEN}✅ Deployment complete!${NC}"
}

main
