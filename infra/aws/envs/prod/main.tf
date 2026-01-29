module "trip_planner_api" {
  source = "../../modules/trip-planner-api"

  project_name = var.project_name
  environment  = "prod"

  vpc_cidr = var.vpc_cidr
  az_count = var.az_count

  image_tag     = var.image_tag
  desired_count = var.desired_count

  jwt_issuer   = var.jwt_issuer
  jwt_audience = var.jwt_audience
  jwt_jwks_url = var.jwt_jwks_url

  public_base_url     = var.public_base_url
  trust_proxy_headers = var.trust_proxy_headers

  domain_name               = var.domain_name
  subject_alternative_names = var.subject_alternative_names
  create_acm_certificate    = var.create_acm_certificate
  certificate_arn           = var.certificate_arn

  alb_ingress_cidr_blocks = var.alb_ingress_cidr_blocks
  log_retention_days      = var.log_retention_days

  db_instance_class            = var.db_instance_class
  db_multi_az                  = var.db_multi_az
  db_backup_retention_days     = var.db_backup_retention_days
  db_deletion_protection       = var.db_deletion_protection
  db_skip_final_snapshot       = var.db_skip_final_snapshot
  db_final_snapshot_identifier = var.db_final_snapshot_identifier

  secrets_kms_key_arn = var.secrets_kms_key_arn

  tags = merge(var.tags, {
    Environment = "prod"
  })
}
