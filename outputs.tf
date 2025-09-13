# 出力値

output "dynamodb_table_name" {
  description = "DynamoDBテーブル名"
  value       = aws_dynamodb_table.inquiry_table.name
}

output "dynamodb_table_arn" {
  description = "DynamoDBテーブルARN"
  value       = aws_dynamodb_table.inquiry_table.arn
}

output "create_answer_lambda_arn" {
  description = "CreateAnswer Lambda関数ARN"
  value       = aws_lambda_function.create_answer.arn
}

output "create_answer_lambda_name" {
  description = "CreateAnswer Lambda関数名"
  value       = aws_lambda_function.create_answer.function_name
}

output "judge_category_lambda_arn" {
  description = "JudgeCategory Lambda関数ARN"
  value       = aws_lambda_function.judge_category.arn
}

output "judge_category_lambda_name" {
  description = "JudgeCategory Lambda関数名"
  value       = aws_lambda_function.judge_category.function_name
}

output "lambda_execution_role_arn" {
  description = "Lambda実行ロールARN"
  value       = aws_iam_role.lambda_execution_role.arn
}

# 既存リソース情報
output "s3_bucket_name" {
  description = "S3バケット名"
  value       = var.s3_bucket_name
}

output "bedrock_knowledge_base_id" {
  description = "BedrockナレッジベースID"
  value       = var.bedrock_knowledge_base_id
}

output "aws_region" {
  description = "AWSリージョン"
  value       = var.aws_region
}
