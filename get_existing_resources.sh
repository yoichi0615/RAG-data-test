#!/bin/bash

# 既存のAWSリソース情報を取得するスクリプト

echo "🔍 既存のAWSリソースを検索中..."
echo ""

# S3バケット一覧
echo "📦 S3バケット一覧:"
aws s3 ls | grep -E "(rag|bedrock|knowledge)" || echo "  関連するバケットが見つかりません"
echo ""

# Bedrockナレッジベース一覧
echo "🧠 Bedrockナレッジベース一覧:"
aws bedrock-agent list-knowledge-bases --query 'knowledgeBaseSummaries[*].[knowledgeBaseId,name,status]' --output table 2>/dev/null || echo "  ナレッジベースが見つかりません（権限不足の可能性）"
echo ""

# 現在のAWSアカウント情報
echo "👤 現在のAWSアカウント:"
aws sts get-caller-identity --query '[Account,Arn]' --output table
echo ""

echo "💡 terraform.tfvarsファイルを作成して、以下の情報を設定してください:"
echo ""
echo "terraform.tfvars:"
echo "s3_bucket_name             = \"your-bucket-name\""
echo "bedrock_knowledge_base_id  = \"your-kb-id\""
echo "bedrock_knowledge_base_arn = \"arn:aws:bedrock:ap-northeast-1:ACCOUNT:knowledge-base/KB-ID\""
echo ""
echo "詳細な設定例は terraform.tfvars.example を参照してください。"
