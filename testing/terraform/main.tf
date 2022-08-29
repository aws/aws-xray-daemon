terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.16"
    }
  }

  required_version = ">=1.2.0"
}

provider "aws" {
  region = var.aws_region
}

resource "random_id" "testing_id" {
  byte_length = 8
}

#########################################
## Create a SSH key pair for EC2 instance.
## Or get an existing one from a S3 bucket
resource "tls_private_key" "ssh_key" {
  count     = var.ssh_key_name == "" ? 1 : 0
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "aws_key_pair" "aws_ssh_key" {
  count      = var.ssh_key_name == "" ? 1 : 0
  key_name   = "keypair-${random_id.testing_id.hex}"
  public_key = tls_private_key.ssh_key[0].public_key_openssh
}

data "aws_s3_bucket_object" "ssh_private_key" {
  count  = var.ssh_key_name != "" ? 1 : 0
  bucket = var.sshkey_s3_bucket
  key    = var.sshkey_s3_private_key
}

locals {
  ssh_key_name        = var.ssh_key_name != "" ? var.ssh_key_name : aws_key_pair.aws_ssh_key[0].key_name
  private_key_content = var.ssh_key_name != "" ? data.aws_s3_bucket_object.ssh_private_key[0].body : tls_private_key.ssh_key[0].private_key_pem
}

# save the private key locally in debug mode.
resource "local_file" "private_key" {
  count    = var.debug ? 1 : 0
  filename = "private_key.pem"
  content  = local.private_key_content
}
#########################################

#########################################
## Provision EC2 instances and run X-Ray Daemon

locals {
  selected_ami         = var.amis[var.testing_ami]
  ami_family           = var.ami_family[local.selected_ami["family"]]
  ami_id               = var.amis[var.testing_ami]["ami_id"]
  instance_type        = lookup(local.selected_ami, "instance_type", local.ami_family["instance_type"])
  login_user           = lookup(local.selected_ami, "login_user", local.ami_family["login_user"])
  connection_type      = local.ami_family["connection_type"]
  ec2_instance_profile = var.ec2_instance_profile
}

resource "aws_instance" "xray_daemon" {
  ami                  = data.aws_ami.ec2_ami
  instance_type        = local.instance_type
  key_name             = local.ssh_key_name
  iam_instance_profile = local.ec2_instance_profile
  tags = {
    Name = "XRayDaemon"
  }
}

resource "null_resource" "wait_for_instance_ready" {
  depends_on = [
    aws_instance.xray_daemon
  ]
  provisioner "remote-exec" {
    inline = [
      local.ami_family["wait_cloud_init"]
    ]
    connection {
      type        = local.connection_type
      user        = local.login_user
      private_key = local.private_key_content
      host        = aws_instance.xray_daemon.public_ip
    }
  }
}

resource "null_resource" "copy_daemon_binary_to_instance" {
  depends_on = [
    null_resource.wait_for_instance_ready
  ]
  provisioner "file" {
    source      = var.daemon_package_local_path
    destination = var.daemon_file_name

    connection {
      type        = local.connection_type
      user        = local.login_user
      private_key = local.private_key_content
      host        = aws_instance.xray_daemon.public_ip
    }
  }
}

resource "null_resource" "install_daemon" {
  depends_on = [
    null_resource.copy_daemon_binary_to_instance
  ]

  provisioner "remote-exec" {
    inline = [
      var.daemon_install_command
    ]
    connection {
      type        = local.connection_type
      user        = local.login_user
      private_key = local.private_key_content
      host        = aws_instance.xray_daemon.public_ip
    }
  }
}

resource "null_resource" "start_daemon" {
  depends_on = [
    null_resource.install_daemon
  ]

  provisioner "remote-exec" {
    inline = [
      var.daemon_start_command,
      "echo sleeping for 10 seconds",
      "for i in {1..10}; do echo 'Sleeping...'$i && sleep 1; done"
    ]
    connection {
      type        = local.connection_type
      user        = local.login_user
      private_key = local.private_key_content
      host        = aws_instance.xray_daemon.public_ip
    }
  }
}

resource "null_resource" "copy_trace_data_to_remote" {
  depends_on = [
    null_resource.start_daemon
  ]

  provisioner "file" {
    source      = var.trace_doc_file_path
    destination = "/home/${local.login_user}/${var.trace_doc_file_name}"

    connection {
      type        = local.connection_type
      user        = local.login_user
      private_key = local.private_key_content
      host        = aws_instance.xray_daemon.public_ip
    }
  }
}

resource "null_resource" "send_trace_data_to_daemon" {
  depends_on = [
    null_resource.copy_trace_data_to_remote
  ]

  provisioner "remote-exec" {
    inline = [
      "#!/bin/bash",
      "cat ${var.trace_doc_file_name} > /dev/udp/127.0.0.1/2000"
    ]
    connection {
      type        = local.connection_type
      user        = local.login_user
      private_key = local.private_key_content
      host        = aws_instance.xray_daemon.public_ip
    }
  }
}
