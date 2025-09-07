terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.1"
    }
  }
}

# AWS Provider - Tokyo Region
provider "aws" {
  region = "ap-northeast-1"
  
  default_tags {
    tags = {
      Project     = var.project_name
      Environment = var.environment
      ManagedBy   = "terraform"
      Region      = "tokyo"
    }
  }
}

# Variables
variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "ap-northeast-1"
}

variable "project_name" {
  description = "Project name"
  type        = string
  default     = "poc-ragbkb"
}

variable "environment" {
  description = "Environment name (only prod is supported)"
  type        = string
  default     = "prod"
  
  validation {
    condition     = var.environment == "prod"
    error_message = "Only 'prod' environment is supported. Dev and staging are not configured for deployment."
  }
}

# S3 Bucket for documents
resource "aws_s3_bucket" "documents_bucket" {
  bucket = "${var.project_name}-documents-${var.environment}-${random_id.bucket_suffix.hex}"
}

resource "random_id" "bucket_suffix" {
  byte_length = 8
}

resource "aws_s3_bucket_versioning" "documents_bucket_versioning" {
  bucket = aws_s3_bucket.documents_bucket.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "documents_bucket_encryption" {
  bucket = aws_s3_bucket.documents_bucket.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "documents_bucket_lifecycle" {
  bucket = aws_s3_bucket.documents_bucket.id

  rule {
    id     = "documents_lifecycle"
    status = "Enabled"

    filter {
      prefix = "documents/"
    }

    transition {
      days          = 30
      storage_class = "STANDARD_IA"
    }

    transition {
      days          = 365
      storage_class = "GLACIER"
    }
  }
}

# DynamoDB Tables
resource "aws_dynamodb_table" "documents_table" {
  name           = "${var.project_name}-documents-${var.environment}"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "id"

  attribute {
    name = "id"
    type = "S"
  }

  attribute {
    name = "status"
    type = "S"
  }

  attribute {
    name = "uploadedAt"
    type = "S"
  }

  global_secondary_index {
    name            = "status-uploadedAt-index"
    hash_key        = "status"
    range_key       = "uploadedAt"
    projection_type = "ALL"
  }

  tags = {
    Name        = "${var.project_name}-documents-${var.environment}"
    Environment = var.environment
  }
}

resource "aws_dynamodb_table" "queries_table" {
  name           = "${var.project_name}-queries-${var.environment}"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "id"

  attribute {
    name = "id"
    type = "S"
  }

  attribute {
    name = "sessionId"
    type = "S"
  }

  attribute {
    name = "timestamp"
    type = "S"
  }

  global_secondary_index {
    name            = "sessionId-timestamp-index"
    hash_key        = "sessionId"
    range_key       = "timestamp"
    projection_type = "ALL"
  }

  ttl {
    attribute_name = "ttl"
    enabled        = true
  }

  tags = {
    Name        = "${var.project_name}-queries-${var.environment}"
    Environment = var.environment
  }
}

resource "aws_dynamodb_table" "responses_table" {
  name           = "${var.project_name}-responses-${var.environment}"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "id"

  attribute {
    name = "id"
    type = "S"
  }

  attribute {
    name = "queryId"
    type = "S"
  }

  global_secondary_index {
    name            = "queryId-index"
    hash_key        = "queryId"
    projection_type = "ALL"
  }

  ttl {
    attribute_name = "ttl"
    enabled        = true
  }

  tags = {
    Name        = "${var.project_name}-responses-${var.environment}"
    Environment = var.environment
  }
}

resource "aws_dynamodb_table" "upload_sessions_table" {
  name           = "${var.project_name}-upload-sessions-${var.environment}"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "id"

  attribute {
    name = "id"
    type = "S"
  }

  ttl {
    attribute_name = "expiresAt"
    enabled        = true
  }

  tags = {
    Name        = "${var.project_name}-upload-sessions-${var.environment}"
    Environment = var.environment
  }
}

# OpenSearch Serverless Collection for Bedrock Knowledge Base
resource "aws_opensearchserverless_security_policy" "bedrock_kb_encryption_policy" {
  name = "ragkb-encryption-${var.environment}"
  type = "encryption"
  policy = jsonencode({
    Rules = [
      {
        Resource = [
          "collection/ragkb-collection-${var.environment}"
        ]
        ResourceType = "collection"
      }
    ]
    AWSOwnedKey = true
  })
}

resource "aws_opensearchserverless_security_policy" "bedrock_kb_network_policy" {
  name = "ragkb-network-${var.environment}"
  type = "network"
  policy = jsonencode([
    {
      Rules = [
        {
          Resource = [
            "collection/ragkb-collection-${var.environment}"
          ]
          ResourceType = "dashboard"
        },
        {
          Resource = [
            "collection/ragkb-collection-${var.environment}"
          ]
          ResourceType = "collection"
        }
      ]
      AllowFromPublic = true
    }
  ])
}

resource "aws_opensearchserverless_collection" "bedrock_kb_collection" {
  name = "ragkb-collection-${var.environment}"
  type = "VECTORSEARCH"

  depends_on = [
    aws_opensearchserverless_security_policy.bedrock_kb_encryption_policy,
    aws_opensearchserverless_security_policy.bedrock_kb_network_policy
  ]
}

# IAM Role for Bedrock Knowledge Base
resource "aws_iam_role" "bedrock_kb_role" {
  name = "${var.project_name}-bedrock-kb-role-${var.environment}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "bedrock.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_policy" "bedrock_kb_policy" {
  name = "${var.project_name}-bedrock-kb-policy-${var.environment}"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.documents_bucket.arn,
          "${aws_s3_bucket.documents_bucket.arn}/*"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "aoss:APIAccessAll"
        ]
        Resource = [
          aws_opensearchserverless_collection.bedrock_kb_collection.arn
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "bedrock:InvokeModel"
        ]
        Resource = [
          "arn:aws:bedrock:ap-northeast-1::foundation-model/*"
        ]
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "bedrock_kb_policy_attachment" {
  role       = aws_iam_role.bedrock_kb_role.name
  policy_arn = aws_iam_policy.bedrock_kb_policy.arn
}

# Data sources for current AWS information
data "aws_region" "current" {}
data "aws_caller_identity" "current" {}

# Outputs
output "aws_region" {
  description = "AWS Region (Tokyo)"
  value       = data.aws_region.current.name
}

output "aws_account_id" {
  description = "AWS Account ID"
  value       = data.aws_caller_identity.current.account_id
}

output "s3_bucket_name" {
  description = "Name of the S3 bucket for documents"
  value       = aws_s3_bucket.documents_bucket.bucket
}

output "documents_table_name" {
  description = "Name of the DynamoDB table for documents"
  value       = aws_dynamodb_table.documents_table.name
}

output "queries_table_name" {
  description = "Name of the DynamoDB table for queries"
  value       = aws_dynamodb_table.queries_table.name
}

output "responses_table_name" {
  description = "Name of the DynamoDB table for responses"
  value       = aws_dynamodb_table.responses_table.name
}

output "upload_sessions_table_name" {
  description = "Name of the DynamoDB table for upload sessions"
  value       = aws_dynamodb_table.upload_sessions_table.name
}

output "opensearch_collection_arn" {
  description = "ARN of the OpenSearch Serverless collection"
  value       = aws_opensearchserverless_collection.bedrock_kb_collection.arn
}

output "opensearch_collection_endpoint" {
  description = "OpenSearch Serverless collection endpoint"
  value       = aws_opensearchserverless_collection.bedrock_kb_collection.collection_endpoint
}

output "bedrock_kb_role_arn" {
  description = "ARN of the IAM role for Bedrock Knowledge Base"
  value       = aws_iam_role.bedrock_kb_role.arn
}