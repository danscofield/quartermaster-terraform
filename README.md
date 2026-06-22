# Terraform Provider for Quartermaster

Manages billets, policies, and billet assignments on a [Quartermaster](https://github.com/dscof/quartermaster) server.

## Provider Configuration

```hcl
provider "quartermaster" {
  endpoint = "https://quartermaster.example.com"
  token    = var.qm_admin_token  # or set QUARTERMASTER_TOKEN env var
  insecure = false               # set true for self-signed certs
}
```

| Attribute | Required | Description |
|-----------|----------|-------------|
| `endpoint` | Yes | Quartermaster server URL |
| `token` | No | Admin JWT. Falls back to `QUARTERMASTER_TOKEN` env var |
| `insecure` | No | Skip TLS verification (default: false) |

## Resources

### `quartermaster_billet`

Creates and manages a billet (the authorization role).

```hcl
resource "quartermaster_billet" "billing_writer" {
  name        = "billing-writer"
  description = "Write access to billing systems"
  tags        = ["workload-only", "sensitivity:high", "team:billing"]

  associated_aws_roles = [
    "arn:aws:iam::123456789012:role/billing-writer",
  ]

  associated_gcp_sas = [
    "billing@my-project.iam.gserviceaccount.com",
  ]
}
```

### `quartermaster_policy`

Creates a raw Cedar policy on a billet. Use this for complex policies that `quartermaster_billet_assignment` can't express.

```hcl
resource "quartermaster_policy" "complex_rule" {
  billet = quartermaster_billet.billing_writer.name

  statement = <<-CEDAR
    permit(
      principal is Quartermaster::AwsRoleIdentity,
      action == Quartermaster::Action::"assumeBillet",
      resource == Quartermaster::Billet::"${quartermaster_billet.billing_writer.name}"
    ) when {
      principal.account_id == "123456789012" &&
      principal.role_name == "billing-service"
    };
  CEDAR

  description = "Billing service in prod account"
}
```

### `quartermaster_billet_assignment`

Simplified declarative billet assignment — generates Cedar under the hood. Use this for common patterns.

The generated Cedar is visible in `terraform show` via the computed `statement` attribute.

---

## Assignment Examples

### AWS — Any identity in an account

```hcl
resource "quartermaster_billet_assignment" "aws_account" {
  billet         = "billing-writer"
  aws_account_id = "526484194718"
  description    = "Any AWS identity in this account gets billing-writer"
}
```

Generated Cedar:
```cedar
permit(principal is Quartermaster::AwsRoleIdentity, action == Quartermaster::Action::"assumeBillet", resource == Quartermaster::Billet::"billing-writer") when { principal.account_id == "526484194718" };
```

### AWS — Specific role in an account

```hcl
resource "quartermaster_billet_assignment" "aws_role" {
  billet         = "billing-writer"
  aws_account_id = "526484194718"
  aws_role_name  = "billing-service"
  description    = "Only the billing-service role"
}
```

Generated Cedar:
```cedar
permit(principal is Quartermaster::AwsRoleIdentity, action == Quartermaster::Action::"assumeBillet", resource == Quartermaster::Billet::"billing-writer") when { principal.account_id == "526484194718" && principal.role_name == "billing-service" };
```

### GCP — Any identity in a project

```hcl
resource "quartermaster_billet_assignment" "gcp_project" {
  billet         = "analytics-reader"
  gcp_project_id = "my-analytics-project"
  description    = "Any GCP identity in the analytics project"
}
```

### OIDC — Group membership

```hcl
resource "quartermaster_billet_assignment" "okta_group" {
  billet        = "billing-writer"
  oidc_group    = "billing-ops"
  oidc_idp_prefix = "okta"
  description   = "Okta billing-ops group members"
}
```

Generated Cedar:
```cedar
permit(principal is Quartermaster::OidcIdentity, action == Quartermaster::Action::"assumeBillet", resource == Quartermaster::Billet::"billing-writer") when { principal.groups.contains("billing-ops") && principal.idp_prefix == "okta" };
```

### OIDC — Any group (any IdP)

```hcl
resource "quartermaster_billet_assignment" "any_idp_group" {
  billet     = "viewer"
  oidc_group = "engineering"
  description = "Anyone in engineering group from any IdP"
}
```

### SPIRE — Exact SPIFFE ID

```hcl
resource "quartermaster_billet_assignment" "exact_spiffe" {
  billet    = "billing-writer"
  spiffe_id = "spiffe://example.com/env/prod/ns/billing/sa/payments-sa"
  description = "Exact workload match"
}
```

### SPIRE — Environment match

```hcl
resource "quartermaster_billet_assignment" "prod_only" {
  billet      = "prod-deployer"
  environment = "prod"
  description = "Only production workloads"
}
```

### SPIRE — Selector match (namespace)

```hcl
resource "quartermaster_billet_assignment" "k8s_namespace" {
  billet   = "billing-writer"
  selector = "k8s:ns:billing"
  description = "All workloads in billing namespace"
}
```

### Compound — Environment + Selector

```hcl
resource "quartermaster_billet_assignment" "prod_billing" {
  billet      = "billing-writer"
  environment = "prod"
  selector    = "k8s:ns:billing"
  description = "Production billing namespace only"
}
```

Generated Cedar:
```cedar
permit(principal is Quartermaster::Workload, action == Quartermaster::Action::"assumeBillet", resource == Quartermaster::Billet::"billing-writer") when { context.selectors.contains("k8s:ns:billing") && principal.environment == "prod" };
```

### Compound — AWS account + environment

```hcl
resource "quartermaster_billet_assignment" "prod_aws" {
  billet         = "prod-writer"
  aws_account_id = "123456789012"
  environment    = "prod"
  description    = "AWS identities from prod account in prod environment"
}
```

---

## Full Example — Multi-Cloud Billing System

```hcl
provider "quartermaster" {
  endpoint = "https://quartermaster.dscof.dev"
  insecure = true
}

# The billet
resource "quartermaster_billet" "billing_writer" {
  name        = "billing-writer"
  description = "Write access to billing systems"
  tags        = ["workload-only", "sensitivity:high", "managed-by:terraform"]

  associated_aws_roles = [aws_iam_role.billing.arn]
  associated_gcp_sas   = [google_service_account.billing.email]
}

# AWS workloads in the billing account
resource "quartermaster_billet_assignment" "aws_billing" {
  billet         = quartermaster_billet.billing_writer.name
  aws_account_id = "123456789012"
  aws_role_name  = "billing-processor"
  description    = "AWS billing processor role"
}

# GCP workloads in the billing project
resource "quartermaster_billet_assignment" "gcp_billing" {
  billet         = quartermaster_billet.billing_writer.name
  gcp_project_id = "billing-prod-123"
  description    = "GCP billing project workloads"
}

# SPIRE workloads in the billing namespace (k8s)
resource "quartermaster_billet_assignment" "k8s_billing" {
  billet      = quartermaster_billet.billing_writer.name
  environment = "prod"
  selector    = "k8s:ns:billing"
  description = "K8s billing namespace in production"
}

# Human operators via Okta
resource "quartermaster_billet_assignment" "human_billing" {
  billet          = quartermaster_billet.billing_writer.name
  oidc_group      = "billing-ops"
  oidc_idp_prefix = "okta"
  description     = "Billing ops team"
}

# Guardrail — prevent non-prod from getting this billet
resource "quartermaster_policy" "guardrail_prod_only" {
  billet = "quartermaster-guardrails"

  statement = <<-CEDAR
    forbid(
      principal,
      action == Quartermaster::Action::"assumeBillet",
      resource == Quartermaster::Billet::"${quartermaster_billet.billing_writer.name}"
    ) when {
      context.source_type == "spire" &&
      principal.environment != "prod"
    };
  CEDAR

  description = "Only prod workloads can assume billing-writer"
}

# AWS IAM role that trusts Quartermaster
resource "aws_iam_role" "billing" {
  name = "billing-writer"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Federated = aws_iam_openid_connect_provider.qm.arn }
      Action    = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        "ForAnyValue:StringEquals" = {
          "quartermaster.dscof.dev:billets" = quartermaster_billet.billing_writer.name
        }
      }
    }]
  })
}

# Register Quartermaster as OIDC provider in AWS
resource "aws_iam_openid_connect_provider" "qm" {
  url             = "https://quartermaster.dscof.dev"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = [var.qm_tls_thumbprint]
}
```

## Cloud IAM Integration

### AWS — Role Assumption via Quartermaster Billets

Register Quartermaster as an OIDC provider in AWS, then create IAM roles that trust specific billets:

```hcl
# Register Quartermaster as an OIDC identity provider
resource "aws_iam_openid_connect_provider" "quartermaster" {
  url             = "https://quartermaster.dscof.dev"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = [var.qm_tls_thumbprint]
}

# IAM role that only holders of "billing-writer" billet can assume
resource "aws_iam_role" "billing_writer" {
  name = "billing-writer"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Federated = aws_iam_openid_connect_provider.quartermaster.arn }
      Action    = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        "StringEquals" = {
          "quartermaster.dscof.dev:aud" = "sts.amazonaws.com"
        }
        "ForAnyValue:StringEquals" = {
          "quartermaster.dscof.dev:billets" = quartermaster_billet.billing_writer.name
        }
      }
    }]
  })
}

# Attach permissions to the role
resource "aws_iam_role_policy_attachment" "billing_writer" {
  role       = aws_iam_role.billing_writer.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonDynamoDBFullAccess"
}

# Store the role ARN on the billet for workload discovery
resource "quartermaster_billet" "billing_writer" {
  name                 = "billing-writer"
  description          = "Write access to billing DynamoDB tables"
  associated_aws_roles = [aws_iam_role.billing_writer.arn]
}
```

Workloads use the flow:
```bash
# 1. Get QM token with billing-writer billet
QM_TOKEN=$(curl -s -X POST https://quartermaster.dscof.dev/token ...)

# 2. Assume the AWS role using the QM token
aws sts assume-role-with-web-identity \
  --role-arn arn:aws:iam::123456789012:role/billing-writer \
  --role-session-name my-session \
  --web-identity-token "$QM_TOKEN"
```

### AWS — Multiple billets can assume the same role

```hcl
resource "aws_iam_role" "shared_reader" {
  name = "shared-reader"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Federated = aws_iam_openid_connect_provider.quartermaster.arn }
      Action    = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        "ForAnyValue:StringEquals" = {
          "quartermaster.dscof.dev:billets" = [
            quartermaster_billet.analytics.name,
            quartermaster_billet.reporting.name,
          ]
        }
      }
    }]
  })
}
```

### GCP — Service Account Impersonation via Quartermaster Billets

Create a Workload Identity Pool that trusts Quartermaster, then bind service accounts to specific billets:

```hcl
# Workload Identity Pool
resource "google_iam_workload_identity_pool" "quartermaster" {
  project                   = var.project
  workload_identity_pool_id = "quartermaster"
  display_name              = "Quartermaster"
}

# OIDC Provider within the pool
resource "google_iam_workload_identity_pool_provider" "quartermaster" {
  project                            = var.project
  workload_identity_pool_id          = google_iam_workload_identity_pool.quartermaster.workload_identity_pool_id
  workload_identity_pool_provider_id = "qm-provider"
  display_name                       = "Quartermaster OIDC"

  oidc {
    issuer_uri = "https://quartermaster.dscof.dev"
  }

  attribute_mapping = {
    "google.subject"       = "assertion.sub"
    "attribute.billets"    = "assertion.billets"
  }
}

# GCP Service Account for billing workloads
resource "google_service_account" "billing" {
  project    = var.project
  account_id = "billing-writer"
}

# Only holders of the "billing-writer" billet can impersonate this SA
resource "google_service_account_iam_binding" "billing_qm" {
  service_account_id = google_service_account.billing.name
  role               = "roles/iam.workloadIdentityUser"

  members = [
    "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.quartermaster.name}/attribute.billets/billing-writer",
  ]
}

# Store the SA on the billet for workload discovery
resource "quartermaster_billet" "billing_writer" {
  name        = "billing-writer"
  description = "Write access to billing systems"

  associated_gcp_sas = [google_service_account.billing.email]
}
```

Workloads use the flow:
```bash
# 1. Get QM token with billing-writer billet (audience = GCP)
QM_TOKEN=$(curl -s -X POST https://quartermaster.dscof.dev/token \
  -d "...&audience=//iam.googleapis.com/projects/PROJECT/locations/global/workloadIdentityPools/quartermaster/providers/qm-provider")

# 2. Exchange QM token for GCP access token
gcloud auth print-access-token --impersonate-service-account=billing-writer@project.iam.gserviceaccount.com
```

### Tying It All Together — One Billet, Both Clouds

```hcl
resource "quartermaster_billet" "billing_writer" {
  name        = "billing-writer"
  description = "Cross-cloud billing access"
  tags        = ["managed-by:terraform", "sensitivity:high"]

  associated_aws_roles = [aws_iam_role.billing_writer.arn]
  associated_gcp_sas   = [google_service_account.billing.email]
}

# One billet assignment — workloads in billing namespace
resource "quartermaster_billet_assignment" "billing_workloads" {
  billet      = quartermaster_billet.billing_writer.name
  selector    = "k8s:ns:billing"
  environment = "prod"
  description = "Production billing workloads get cross-cloud access"
}
```

A single workload in the `billing` namespace can now assume the AWS role OR impersonate the GCP SA — both gated by the same Quartermaster billet, managed in one Terraform apply.

## Escape Hatch

For anything `quartermaster_billet_assignment` can't express, use `quartermaster_policy` with raw Cedar:

```hcl
resource "quartermaster_policy" "time_based" {
  billet = "incident-escalation"

  statement = <<-CEDAR
    permit(
      principal is Quartermaster::OidcIdentity,
      action == Quartermaster::Action::"assumeBillet",
      resource == Quartermaster::Billet::"incident-escalation"
    ) when {
      principal.groups.contains("oncall") &&
      context.environment == "prod"
    };
  CEDAR

  description = "Oncall team can escalate in prod"
}
```

## Building

```bash
go build -o terraform-provider-quartermaster
```

## Local Development

```bash
# Override provider path for local testing
cat > ~/.terraformrc <<EOF
provider_installation {
  dev_overrides {
    "dscof/quartermaster" = "/path/to/qm-terraform"
  }
  direct {}
}
EOF
```
