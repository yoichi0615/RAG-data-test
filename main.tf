# Terraform設定
terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.49"
    }
  }
}

# AWSプロバイダー設定
provider "aws" {
  region = var.aws_region
}

# S3バケット (既存リソースをインポート)
resource "aws_s3_bucket" "s3_bucket" {
  bucket = var.s3_bucket_name
}

# Bedrockナレッジベース (既存リソースをインポート)
resource "aws_bedrockagent_knowledge_base" "bedrock_knowledge_base" {
  name     = "knowledge-base-quick-start-c53hm"
  role_arn = "arn:aws:iam::706057796583:role/service-role/AmazonBedrockExecutionRoleForKnowledgeBase_c53hm"

  knowledge_base_configuration {
    type = "VECTOR"
    vector_knowledge_base_configuration {
      embedding_model_arn = "arn:aws:bedrock:ap-northeast-1::foundation-model/cohere.embed-multilingual-v3"
      embedding_model_configuration {
        bedrock_embedding_model_configuration {
          embedding_data_type = "FLOAT32"
        }
      }
    }
  }

  storage_configuration {
    type = "OPENSEARCH_SERVERLESS"
    opensearch_serverless_configuration {
      collection_arn    = "arn:aws:aoss:ap-northeast-1:706057796583:collection/wzwe2ugsjfsvmt7pws3k"
      vector_index_name = "bedrock-knowledge-base-default-index"
      field_mapping {
        vector_field   = "bedrock-knowledge-base-default-vector"
        text_field     = "AMAZON_BEDROCK_TEXT"
        metadata_field = "AMAZON_BEDROCK_METADATA"
      }
    }
  }
}

# 1. DynamoDB テーブル - InquiryTable
resource "aws_dynamodb_table" "inquiry_table" {
  name         = var.dynamodb_table_name
  billing_mode = "PAY_PER_REQUEST" # 最低スペック
  hash_key     = "id"

  attribute {
    name = "id"
    type = "S"
  }

  tags = {
    Name        = var.dynamodb_table_name
    Environment = "learning"
  }
}

# 2. Lambda実行用IAMロール
resource "aws_iam_role" "lambda_execution_role" {
  name = "${var.project_name}-lambda-execution-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name        = "${var.project_name}-lambda-execution-role"
    Environment = "learning"
  }
}

# 3. Lambda実行用IAMポリシー
resource "aws_iam_role_policy" "lambda_execution_policy" {
  name = "${var.project_name}-lambda-execution-policy"
  role = aws_iam_role.lambda_execution_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      # CloudWatch Logs
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "arn:aws:logs:*:*:*"
      },
      # DynamoDB
      {
        Effect = "Allow"
        Action = [
          "dynamodb:GetItem",
          "dynamodb:PutItem",
          "dynamodb:UpdateItem",
          "dynamodb:Query",
          "dynamodb:Scan"
        ]
        Resource = [
          aws_dynamodb_table.inquiry_table.arn,
          "${aws_dynamodb_table.inquiry_table.arn}/index/*"
        ]
      },
      # S3 (既存バケット)
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:ListBucket"
        ]
        Resource = [
          "arn:aws:s3:::${var.s3_bucket_name}",
          "arn:aws:s3:::${var.s3_bucket_name}/*"
        ]
      },
      # Bedrock
      {
        Effect = "Allow"
        Action = [
          "bedrock:InvokeModel",
          "bedrock:RetrieveAndGenerate",
          "bedrock:Retrieve"
        ]
        Resource = [
          "arn:aws:bedrock:${var.aws_region}::foundation-model/anthropic.claude-3-haiku-20240307-v1:0",
          var.bedrock_knowledge_base_arn
        ]
      }
    ]
  })
}



# 5. CreateAnswer Lambda関数 (Go)
resource "aws_lambda_function" "create_answer" {
  function_name = "${var.project_name}-create-answer"
  role          = aws_iam_role.lambda_execution_role.arn
  handler       = "bootstrap"
  runtime       = "provided.al2"
  architectures = ["arm64"]
  timeout       = 60
  memory_size   = 256

  filename         = "${path.module}/go-lambdas/create-answer/function.zip"
  source_code_hash = filebase64sha256("${path.module}/go-lambdas/create-answer/function.zip")

  environment {
    variables = {
      INQUIRY_TABLE_NAME        = aws_dynamodb_table.inquiry_table.name
      S3_BUCKET_NAME            = var.s3_bucket_name
      BEDROCK_KNOWLEDGE_BASE_ID = var.bedrock_knowledge_base_id
    }
  }

  depends_on = [
    aws_iam_role_policy.lambda_execution_policy,
    aws_cloudwatch_log_group.create_answer_log_group,
  ]

  tags = {
    Name        = "${var.project_name}-create-answer"
    Environment = "learning"
  }
}

# 6. JudgeCategory Lambda関数 (Go)
resource "aws_lambda_function" "judge_category" {
  function_name = "${var.project_name}-judge-category"
  role          = aws_iam_role.lambda_execution_role.arn
  handler       = "bootstrap"
  runtime       = "provided.al2"
  architectures = ["arm64"]
  timeout       = 30
  memory_size   = 256

  filename         = "${path.module}/go-lambdas/judge-category/function.zip"
  source_code_hash = filebase64sha256("${path.module}/go-lambdas/judge-category/function.zip")

  environment {
    variables = {
      INQUIRY_TABLE_NAME = aws_dynamodb_table.inquiry_table.name
    }
  }

  depends_on = [
    aws_iam_role_policy.lambda_execution_policy,
    aws_cloudwatch_log_group.judge_category_log_group,
  ]

  tags = {
    Name        = "${var.project_name}-judge-category"
    Environment = "learning"
  }
}

# 6. CloudWatch Log Groups
resource "aws_cloudwatch_log_group" "create_answer_log_group" {
  name              = "/aws/lambda/${var.project_name}-create-answer"
  retention_in_days = 7 # 最低スペック

  tags = {
    Name        = "${var.project_name}-create-answer-logs"
    Environment = "learning"
  }
}

resource "aws_cloudwatch_log_group" "judge_category_log_group" {
  name              = "/aws/lambda/${var.project_name}-judge-category"
  retention_in_days = 7 # 最低スペック

  tags = {
    Name        = "${var.project_name}-judge-category-logs"
    Environment = "learning"
  }
}
