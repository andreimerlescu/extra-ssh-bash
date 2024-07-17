terraform {
    backend "http" {
        address = "https://gitlab.com/api/v4/projects/PROJECT_ID/terraform/state/docker-cluster-tfstate"
        lock_address = "https://gitlab.com/api/v4/projects/PROJECT_ID/terraform/state/docker-cluster-tfstate/lock"
        lock_method = "POST"
        unlock_address = "https://gitlab.com/api/v4/projects/PROJECT_ID/terraform/state/docker-cluster-tfstate/lock"
        unlock_method = "DELETE"
        skip_cert_verification = true
        retry_max = 3
        retry_wait_min = 12
        retry_wait_max = 30
    }
    required_providers {
        aws = {
            source = "hashicorp/aws"
            version = "~> 5.0"
        }
        tls = {
            source = "hashicorp/tls"
            version = "~> 4.0"
        }
    }
}
provider "aws" {
    region = "us-east-1" # CHANGE TO YOUR PREFERRED AWS AVAILABILITY ZONE
}

resource "aws_key_pair" "docker_cluster_sshkey" {
    key_name = "docker_cluster_sshkey"
    public_key = "REPLACE_WITH_SSH_PUBLIC_KEY"
}

variable "docker_instance_count" {
  type = number
  default = 3
}

resource "aws_instance" "docker_member" {
    count = var.docker_instance_count
    ami = "ami-04a81a99f5ec58529" # Ubuntu 24.04 LTS
    instance_type = "t3.nano" # smallest instance size possible
    subnet_id = "subnet-REPLACEME" # ADD YOUR AWS SUBNET HERE
    availability_zone = "us-east-1d" # CHANGE TO YOUR AWS REGION
    key_name = aws_key_pair.docker_cluster_sshkey.key_name
    monitoring = true
    associate_public_ip_address = true # Security Group Protects Endpoint
    vpc_security_group_ids = [
        #SG1-REPLACEME
        #SG2-REPLACEME
        #SG3-REPLACEME
        #SG4-REPLACEME
        #SG5-REPLACEME
    ]
    tags = {
      Name = "docker-cluster-member-${count.index}"
      Author = "Terraform"
    }
    root_block_device {
      volume_size = 36
      volume_type = "gp3"
    }
    user_data = <<-EOT
#!/bin/bash
sudo apt-get update
sudo apt-get upgrade -y

# Install bash
safe_exit() {
  local msg="$${1:-UnexpectedError}"
  echo "$${msg}"
  exit 1
}

mkdir -p "/tmp/bash-install"
cd "/tmp/bash-install" || safe_exit "Cannot access /tmp/bash-install"
sudo apt-get -y build-essential curl htop jq
curl -O http://ftp.gnu.org/gnu/bash/bash-5.2.tar.gz
tar xvf bash-5.*.tar.gz
cd bash-5.*/ || safe_exit "Cannot find expanded bash archive"
./configure
make
sudo make install

# Install docker
sudo apt-get update
sudo apt-get install ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc

# Add the repository to Apt sources:
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update -y
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo systemctl daemon-reload
sudo systemctl enable docker
sudo systemctl start docker
sudo groupadd docker
sudo usermod -a -G docker ubuntu
EOT
    user_data_replace_on_change = false
}

output "instance_ids" {
    value = aws_instance.docker_member[*].id
}

output "private_ips" {
    value = aws_instance.docker_member[*].private_ip
}

output "public_ips" {
  value = aws_instance.docker_member[*].public_ip
}

