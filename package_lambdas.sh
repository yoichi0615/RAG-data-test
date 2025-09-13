#!/bin/bash

# スクリプトが失敗した場合に終了する
set -e

echo "📦 Packaging Go Lambda functions..."

# JudgeCategory Lambda
echo "Building judge-category Lambda..."
cd go-lambdas/judge-category
GOOS=linux GOARCH=arm64 go build -o bootstrap main.go
zip function.zip bootstrap
rm bootstrap
cd ../../

# CreateAnswer Lambda
echo "Building create-answer Lambda..."
cd go-lambdas/create-answer
GOOS=linux GOARCH=arm64 go build -o bootstrap main.go
zip function.zip bootstrap
rm bootstrap
cd ../../

echo "✅ Go Lambda functions packaged successfully:"
echo "  - go-lambdas/judge-category/function.zip"
echo "  - go-lambdas/create-answer/function.zip"
