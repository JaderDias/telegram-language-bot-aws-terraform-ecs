variable "prefix" {
  description = "prefix prepended to names of all resources created"
  default     = "telegram-bot-go-fargate"
}

variable "port" {
  description = "port the container exposes, that the load balancer should forward port 80 to"
  default     = 4000
  type        = number
}

variable "region" {
  description = "selects the aws region to apply these services to"
  default     = "eu-central-1"
}

variable "source_path" {
  description = "source path for project"
  default     = "./golang/bot"
}

variable "tag" {
  description = "tag to use for our new docker image"
  default     = "latest"
}

variable "envvars" {
  type        = map(string)
  description = "variables to set in the environment of the container"
  default = {
  }
}

variable "telegram_bot_token" {
  type = string
}

resource "random_pet" "this" {
  length = 2
}

provider "aws" {
  region = var.region
}

resource "aws_ecs_cluster" "staging" {
  name = "${var.prefix}-cluster"
}

data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

data "aws_caller_identity" "current" {}

output "account_id" {
  value = data.aws_caller_identity.current.account_id
}

resource "aws_security_group" "ecs_tasks" {
  name        = "${var.prefix}-tasks-sg"
  description = "Does not allow inbound"

  ingress {
    protocol    = "-1"
    from_port   = 0
    to_port     = 0
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    protocol    = "-1"
    from_port   = 0
    to_port     = 0
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_ecr_repository" "repo" {
  name = "${var.prefix}/runner"
}

resource "aws_ecr_lifecycle_policy" "repo-policy" {
  repository = aws_ecr_repository.repo.name

  policy = <<EOF
{
  "rules": [
    {
      "rulePriority": 1,
      "description": "Keep image deployed with tag latest",
      "selection": {
        "tagStatus": "tagged",
        "tagPrefixList": ["latest"],
        "countType": "imageCountMoreThan",
        "countNumber": 1
      },
      "action": {
        "type": "expire"
      }
    },
    {
      "rulePriority": 2,
      "description": "Keep last 2 any images",
      "selection": {
        "tagStatus": "any",
        "countType": "imageCountMoreThan",
        "countNumber": 2
      },
      "action": {
        "type": "expire"
      }
    }
  ]
}
EOF
}

data "aws_iam_policy_document" "ecs_task_execution_role" {
  version = "2012-10-17"
  statement {
    sid     = ""
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "ecs_task_execution_role" {
  name               = "${var.prefix}-execution-role"
  assume_role_policy = data.aws_iam_policy_document.ecs_task_execution_role.json
}

resource "aws_iam_role_policy_attachment" "ecs_task_execution_role" {
  role       = aws_iam_role.ecs_task_execution_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_s3_bucket" "bucket" {
  bucket = "my-bucket-${random_pet.this.id}"
}

resource "aws_s3_object" "object" {
  bucket  = aws_s3_bucket.bucket.id
  key     = "telegram_bot_token"
  content = var.telegram_bot_token
  etag    = md5(var.telegram_bot_token)
}

resource "aws_ecs_task_definition" "service" {
  family             = "${var.prefix}-task-family"
  network_mode       = "awsvpc"
  execution_role_arn = aws_iam_role.ecs_task_execution_role.arn
  cpu                = 256 # 256 is the minimum and reperesents 0.25 vCPUs
  memory             = 512 # 512 MB is the minimum
  runtime_platform {
    cpu_architecture        = "ARM64"
    operating_system_family = "LINUX"
  }
  requires_compatibilities = ["FARGATE"]
  container_definitions = jsonencode([{
    name      = "${var.prefix}-task-service"
    command   = ["${var.telegram_bot_token}"]
    image     = "${aws_ecr_repository.repo.repository_url}:latest"
    essential = true
    logConfiguration = {
      logDriver = "awslogs"
      options = {
        awslogs-region        = "${var.region}"
        awslogs-stream-prefix = "${var.prefix}-service"
        awslogs-group         = "${var.prefix}-log-group"
      }
    }
    portMappings = [
      {
        containerPort = var.port
        hostPort      = var.port
        protocol      = "tcp"
      }
    ]
    ulimits = [
      {
        name      = "nofile"
        softLimit = 65536
        hardLimit = 65536
      }
    ]
    mountPoints = []
    cpu         = 256 # 256 is the minimum and reperesents 0.25 vCPUs
    memory      = 512 # 512 MB is the minimum
    volumesFrom = []
  }])
  tags = {
    Environment = "staging"
    Application = "${var.prefix}-app"
  }
}

resource "aws_ecs_service" "staging" {
  name            = "${var.prefix}-service"
  cluster         = aws_ecs_cluster.staging.id
  task_definition = aws_ecs_task_definition.service.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    security_groups  = [aws_security_group.ecs_tasks.id]
    subnets          = data.aws_subnets.default.ids
    assign_public_ip = true
  }

  depends_on = [aws_iam_role_policy_attachment.ecs_task_execution_role]

  tags = {
    Environment = "staging"
    Application = "${var.prefix}-app"
  }
}

resource "aws_cloudwatch_log_group" "dummyapi" {
  name = "${var.prefix}-log-group"

  tags = {
    Environment = "staging"
    Application = "${var.prefix}-app"
  }
}

resource "null_resource" "generate_dictionary" {
  triggers = {
    always_run = "${timestamp()}"
  }
  provisioner "local-exec" {
    command     = "./generate_dictionary.sh ${var.source_path}"
    interpreter = ["bash", "-c"]
  }
}

resource "null_resource" "push_docker_image" {
  triggers = {
    always_run = "${timestamp()}"
  }
  provisioner "local-exec" {
    command     = "./push_docker_image.sh ${var.source_path} ${aws_ecr_repository.repo.repository_url} ${var.tag} ${data.aws_caller_identity.current.account_id}"
    interpreter = ["bash", "-c"]
  }
  depends_on = [
    resource.null_resource.generate_dictionary
  ]
}