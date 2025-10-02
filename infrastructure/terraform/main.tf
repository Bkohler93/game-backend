provider "aws" {
  region = "us-east-1"
}

data "aws_ami" "ubuntu" {
  most_recent = true

  filter {
    name = "name"
    values = ["ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-*"]
  }

  owners = ["099720109477"] # Canonical
}

resource "aws_instance" "app_server" {
  ami           = data.aws_ami.ubuntu.id
  instance_type = "t2.micro"
  
  subnet_id = aws_subnet.public_1.id # Or aws_subnet.public_2.id, depending on your preference
  vpc_security_group_ids = [aws_security_group.app.id,aws_security_group.allow_ssh.id]
  key_name      = aws_key_pair.ec2_instance.key_name

  iam_instance_profile = aws_iam_instance_profile.ec2_instance_profile.name

  tags = {
    Name = "game-backend"
  }
}

resource "aws_iam_role" "ec2_role" {
  name = "ec2-ecr-access-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Service = "ec2.amazonaws.com"
      }
      Action = "sts:AssumeRole"
    }]
  })
}

# 2. Attach policy to allow ECR access and basic EC2 permissions
resource "aws_iam_role_policy_attachment" "ecr_access" {
  role       = aws_iam_role.ec2_role.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
}

# Optional: Attach other policies if needed, e.g. S3 access, CloudWatch logs, etc.

# 3. Create Instance Profile (required for EC2 to use the role)
resource "aws_iam_instance_profile" "ec2_instance_profile" {
  name = "ec2-instance-profile"
  role = aws_iam_role.ec2_role.name
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "main-vpc"
  }
}

resource "aws_security_group" "allow_ssh" {
  name        = "allow_ssh"
  description = "Allow SSH from my IP"
  vpc_id      = aws_vpc.main.id

  ingress {
    description      = "SSH from my IP"
    from_port        = 22
    to_port          = 22
    protocol         = "tcp"
    cidr_blocks      = ["67.237.164.19/32"] # Replace with your public IP or use "0.0.0.0/0" for open access (not recommended)
  }

  egress {
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
  }

  tags = {
    Name = "allow_ssh"
  }
}


resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "main-igw"
  }
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }

  tags = {
    Name = "public-route-table"
  }
}

resource "aws_route_table_association" "public_1" {
  subnet_id      = aws_subnet.public_1.id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table_association" "public_2" {
  subnet_id      = aws_subnet.public_2.id
  route_table_id = aws_route_table.public.id
}

resource "aws_subnet" "public_1" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.1.0/24"
  availability_zone = "us-east-1a"
  map_public_ip_on_launch = true

  tags = {
    Name = "public-subnet-1"
  }
}



resource "aws_subnet" "public_2" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.2.0/24"
  availability_zone = "us-east-1b"
  map_public_ip_on_launch = true

  tags = {
    Name = "public-subnet-2"
  }
}

resource "aws_security_group" "alb" {
  name        = "alb-sg"
  description = "Allow traffic to the ALB"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_lb" "web" {
  name               = "web-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb.id]
  subnets            = [aws_subnet.public_1.id, aws_subnet.public_2.id]

  enable_deletion_protection = false

  tags = {
    Environment = "production"
  }
}

resource "aws_security_group" "app" {
  name   = "app-sg"
  vpc_id = aws_vpc.main.id

  ingress {
    from_port       = 8089
    to_port         = 8089
    protocol        = "tcp"
    security_groups = [aws_security_group.alb.id]  # Allow ALB traffic
  }

  ingress {
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.alb.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "app-security-group"
  }
}

resource "aws_lb_target_group" "websocket" {
  name     = "websocket-target-group"
  port     = 8089
  protocol = "HTTP"
  vpc_id   = aws_vpc.main.id

  health_check {
    path                = "/ws/health"
    healthy_threshold   = 2
    unhealthy_threshold = 2
  }

  tags = {
    TargetGroup = "websocket"
  }
}

resource "aws_lb_target_group" "auth" {
  name     = "auth-target-group"
  port     = 8080
  protocol = "HTTP"
  vpc_id   = aws_vpc.main.id

  health_check {
    path                = "/auth/health"
    healthy_threshold   = 2
    unhealthy_threshold = 2
  }

  tags = {
    TargetGroup = "auth"
  }
}

resource "aws_lb_target_group_attachment" "websocket" {
  target_group_arn = aws_lb_target_group.websocket.arn
  target_id        = aws_instance.app_server.id
  port             = 8089
}

resource "aws_lb_target_group_attachment" "auth" {
  target_group_arn = aws_lb_target_group.auth.arn
  target_id        = aws_instance.app_server.id
  port             = 8080
}


resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.web.arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type = "fixed-response"
    fixed_response {
      content_type = "text/plain"
      message_body = "Not found"
      status_code  = "404"
    }
  }
}

resource "aws_route53_zone" "primary" {
  name = var.domain_name
}

resource "aws_acm_certificate" "tls_cert" {
  domain_name       = var.domain_name
  validation_method = "DNS"

  tags = {
    Name = "TLS Certificate"
  }

  lifecycle {
    create_before_destroy = true
  }
}


resource "aws_route53_record" "cert_validation" {
  for_each = {
    for dvo in aws_acm_certificate.tls_cert.domain_validation_options : dvo.domain_name => {
      name  = dvo.resource_record_name
      type  = dvo.resource_record_type
      record = dvo.resource_record_value
    }
  }

  zone_id = aws_route53_zone.primary.zone_id
  name    = each.value.name
  type    = each.value.type
  ttl     = 60
  records = [each.value.record]
}

resource "aws_acm_certificate_validation" "cert" {
  certificate_arn         = aws_acm_certificate.tls_cert.arn
  validation_record_fqdns = values(aws_route53_record.cert_validation)[*].fqdn
}


resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.web.arn
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-2016-08"
  certificate_arn   = aws_acm_certificate_validation.cert.certificate_arn
  
  lifecycle {
    create_before_destroy = true 
  }

  default_action {
    type = "fixed-response"
    fixed_response {
      content_type = "text/plain"
      message_body = "Not found"
      status_code  = "404"
    }
  }
}

resource "aws_lb_listener_rule" "websocket" {
  listener_arn = aws_lb_listener.http.arn
  priority     = 100

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.websocket.arn
  }

  condition {
    path_pattern {
      values = ["/ws*"]
    }
  }
}

resource "aws_lb_listener_rule" "auth" {
  listener_arn = aws_lb_listener.http.arn
  priority     = 101

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.auth.arn
  }

  condition {
    path_pattern {
      values = ["/auth*"]
    }
  }
}

resource "aws_lb_listener_rule" "websocket_https" {
  listener_arn = aws_lb_listener.https.arn
  priority     = 102

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.websocket.arn
  }

  condition {
    path_pattern {
      values = ["/ws*"]
    }
  }
}

resource "aws_lb_listener_rule" "auth_https" {
  listener_arn = aws_lb_listener.https.arn
  priority     = 103

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.auth.arn
  }

  condition {
    path_pattern {
      values = ["/auth*"]
    }
  }
}



resource "aws_route53_record" "app" {
  zone_id = aws_route53_zone.primary.zone_id
  name    = var.domain_name
  type    = "A"

  alias {
    name                   = aws_lb.web.dns_name
    zone_id                = aws_lb.web.zone_id
    evaluate_target_health = true
  }
}

resource "tls_private_key" "ec2_instance" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "aws_key_pair" "ec2_instance" {
  key_name   = "ec2-key"
  public_key = tls_private_key.ec2_instance.public_key_openssh
}

resource "local_file" "private_key" {
  content  = tls_private_key.ec2_instance.private_key_pem
  filename = "${path.module}/ec2_key.pem"
  file_permission = "0600"
}

