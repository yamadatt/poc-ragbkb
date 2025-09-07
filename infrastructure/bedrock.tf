# Bedrock Knowledge Base Configuration for Tokyo Region
# Data Access Policy for OpenSearch Serverless
resource "aws_opensearchserverless_access_policy" "bedrock_kb_access_policy" {
  name = "${var.project_name}-kb-access-policy-${var.environment}"
  type = "data"
  
  policy = jsonencode([
    {
      Rules = [
        {
          Resource = [
            "collection/ragkb-collection-${var.environment}"
          ]
          Permission = [
            "aoss:CreateCollectionItems",
            "aoss:DeleteCollectionItems", 
            "aoss:UpdateCollectionItems",
            "aoss:DescribeCollectionItems"
          ]
          ResourceType = "collection"
        },
        {
          Resource = [
            "index/ragkb-collection-${var.environment}/*"
          ]
          Permission = [
            "aoss:CreateIndex",
            "aoss:DeleteIndex",
            "aoss:UpdateIndex", 
            "aoss:DescribeIndex",
            "aoss:ReadDocument",
            "aoss:WriteDocument"
          ]
          ResourceType = "index"
        }
      ]
      Principal = [
        aws_iam_role.bedrock_kb_role.arn,
        "arn:aws:iam::449671225256:user/yamada"
      ]
    }
  ])
}

# Create Vector Index before Knowledge Base

# Bedrock Knowledge Base
resource "aws_bedrockagent_knowledge_base" "rag_knowledge_base" {
  name         = "${var.project_name}-kb-${var.environment}"
  description  = "RAG Knowledge Base for document retrieval and generation"
  role_arn     = aws_iam_role.bedrock_kb_role.arn

  knowledge_base_configuration {
    vector_knowledge_base_configuration {
      embedding_model_arn = "arn:aws:bedrock:ap-northeast-1::foundation-model/amazon.titan-embed-text-v2:0"
    }
    type = "VECTOR"
  }

  storage_configuration {
    type = "OPENSEARCH_SERVERLESS"
    opensearch_serverless_configuration {
      collection_arn    = aws_opensearchserverless_collection.bedrock_kb_collection.arn
      vector_index_name = "${var.project_name}-vector-index"
      field_mapping {
        vector_field   = "vector"
        text_field     = "text"
        metadata_field = "metadata"
      }
    }
  }

  tags = {
    Name        = "${var.project_name}-knowledge-base-${var.environment}"
    Environment = var.environment
    Purpose     = "RAG"
    Region      = "ap-northeast-1"
  }

  depends_on = [
    aws_opensearchserverless_collection.bedrock_kb_collection,
    aws_opensearchserverless_access_policy.bedrock_kb_access_policy,
    aws_iam_role_policy_attachment.bedrock_kb_policy_attachment
  ]
}

# Bedrock Data Source
resource "aws_bedrockagent_data_source" "rag_data_source" {
  knowledge_base_id = aws_bedrockagent_knowledge_base.rag_knowledge_base.id
  name              = "${var.project_name}-data-source-${var.environment}"
  description       = "S3 data source for RAG system documents"

  data_source_configuration {
    type = "S3"
    s3_configuration {
      bucket_arn = aws_s3_bucket.documents_bucket.arn
      inclusion_prefixes = ["documents/"]
    }
  }

  vector_ingestion_configuration {
    chunking_configuration {
      chunking_strategy = "FIXED_SIZE"
      fixed_size_chunking_configuration {
        max_tokens         = 512
        overlap_percentage = 20
      }
    }
  }


  depends_on = [
    aws_bedrockagent_knowledge_base.rag_knowledge_base
  ]
}

# Output for Knowledge Base
output "knowledge_base_id" {
  description = "Bedrock Knowledge Base ID"
  value       = aws_bedrockagent_knowledge_base.rag_knowledge_base.id
}

output "knowledge_base_arn" {
  description = "Bedrock Knowledge Base ARN"
  value       = aws_bedrockagent_knowledge_base.rag_knowledge_base.arn
}

output "data_source_id" {
  description = "Bedrock Data Source ID"
  value       = aws_bedrockagent_data_source.rag_data_source.data_source_id
}
