# telegram-bot-aws-terraform-ecs-fargate-go
Telegram bot hosted in Amazon Web Services using Elastic Container Service and Fargate

## Prerequisites

1. Docker installed and running
2. Terraform

## Deployment

1. `terraform init`
2. `terraform apply --var "telegram_bot_token=YOUR_TOKEN_HERE"`