# IAM Policy for OpenSearch Serverless access for yamada user
resource "aws_iam_policy" "opensearch_user_policy" {
  name        = "OpenSearchServerlessUserPolicy"
  description = "Policy for yamada user to access OpenSearch Serverless"
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "aoss:*",
          "opensearch:*"
        ]
        Resource = "*"
      }
    ]
  })
}

# Attach policy to yamada user
resource "aws_iam_user_policy_attachment" "opensearch_user_policy_attachment" {
  user       = "yamada"
  policy_arn = aws_iam_policy.opensearch_user_policy.arn
}