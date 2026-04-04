provider "aws" {
  region = var.aws_region

  default_tags {
    tags = merge(
      {
        Project    = var.project_name
        ManagedBy  = "terraform"
        Repository = "${var.github_org}/${var.github_repo}"
      },
      var.tags
    )
  }
}

resource "aws_s3_bucket" "tf_state" {
  bucket        = var.tfstate_bucket_name
  force_destroy = var.tfstate_bucket_force_destroy
}

resource "aws_s3_bucket_versioning" "tf_state" {
  bucket = aws_s3_bucket.tf_state.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "tf_state" {
  bucket = aws_s3_bucket.tf_state.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "tf_state" {
  bucket = aws_s3_bucket.tf_state.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_ownership_controls" "tf_state" {
  bucket = aws_s3_bucket.tf_state.id

  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_dynamodb_table" "tf_lock" {
  name         = var.tfstate_lock_table_name
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "LockID"

  attribute {
    name = "LockID"
    type = "S"
  }
}

resource "aws_iam_openid_connect_provider" "github" {
  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = var.github_oidc_thumbprints
}

data "aws_iam_policy_document" "github_assume_role" {
  for_each = var.github_subjects

  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.github.arn]
    }

    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }

    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values   = each.value
    }
  }
}

resource "aws_iam_role" "github_deploy" {
  for_each = var.github_subjects

  name               = "${var.project_name}-${each.key}-deploy"
  assume_role_policy = data.aws_iam_policy_document.github_assume_role[each.key].json
}

locals {
  deploy_role_policy_bindings = {
    for pair in setproduct(keys(var.github_subjects), var.deploy_role_policy_arns) :
    "${pair[0]}-${md5(pair[1])}" => {
      env        = pair[0]
      policy_arn = pair[1]
    }
  }
}

resource "aws_iam_role_policy_attachment" "github_deploy" {
  for_each   = local.deploy_role_policy_bindings
  role       = aws_iam_role.github_deploy[each.value.env].name
  policy_arn = each.value.policy_arn
}
