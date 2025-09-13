# 変数定義

variable "aws_region" {
  description = "AWS リージョン"
  type        = string
  default     = "ap-northeast-1"
}

variable "project_name" {
  description = "プロジェクト名（リソース名のプレフィックス）"
  type        = string
  default     = "rag-system"
}

# 既存リソースの情報
variable "s3_bucket_name" {
  description = "既存のS3バケット名"
  type        = string
}

variable "bedrock_knowledge_base_id" {
  description = "既存のBedrockナレッジベースID"
  type        = string
}

variable "bedrock_knowledge_base_arn" {
  description = "既存のBedrockナレッジベースARN"
  type        = string
}

variable "dynamodb_table_name" {
  description = "DynamoDBテーブル名"
  type        = string
  default     = "InquiryTable"
}
