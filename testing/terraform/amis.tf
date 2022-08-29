variable "ami_family" {
  default = {
    debian = {
      login_user      = "ubuntu"
      instance_type   = "t2.micro"
      connection_type = "ssh"
      wait_cloud_init = "for i in {1..300}; do [ ! -f /var/lib/cloud/instance/boot-finished ] && echo 'Waiting for cloud-init...'$i && sleep 1 || break; done"
    }
    linux = {
      login_user      = "ec2-user"
      instance_type   = "t2.micro"
      connection_type = "ssh"
      wait_cloud_init = "for i in {1..300}; do [ ! -f /var/lib/cloud/instance/boot-finished ] && echo 'Waiting for cloud-init...'$i && sleep 1 || break; done"
    }
  }
}

variable "amis" {
  default = {
    ubuntu18 = {
      os_family          = "ubuntu"
      ami_search_pattern = "ubuntu/images/hvm-ssd/ubuntu-bionic-18.04-amd64-server*"
      ami_owner          = "099720109477"
      ami_id             = "ami-02da34c96f69d525c"
      ami_product_code   = []
      family             = "debian"
      arch               = "amd64"
      login_user         = "ubuntu"
    }
    amazonlinux2 = {
      os_family          = "amazon_linux"
      ami_search_pattern = "amzn2-ami-hvm-2.0.????????.?-x86_64-gp2"
      ami_owner          = "amazon"
      ami_id             = "ami-0d08ef957f0e4722b"
      ami_product_code   = []
      family             = "linux"
      arch               = "amd64"
      login_user         = "ec2-user"
    }
    redhat8 = {
      os_family          = "redhat"
      ami_search_pattern = "RHEL-8.6.0_HVM*"
      ami_owner          = "309956199498"
      ami_id             = "ami-087c2c50437d0b80d"
      ami_product_code   = []
      family             = "linux"
      arch               = "amd64"
    }
  }
}

data "aws_ami" "ec2_ami" {
  most_recent = true

  filter {
    name   = "name"
    values = [var.amis[var.testing_ami]["ami_search_pattern"]]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  owners = [var.amis[var.testing_ami]["ami_owner"]]
}