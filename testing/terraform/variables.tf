variable "testing_ami" {
  default = "amazonlinux2"
}

variable "daemon_file_name" {
  default = ""
}

variable "daemon_install_command" {
  default = ""
}

variable "daemon_start_command" {
  default = ""
}

variable "daemon_package_local_path" {
  default = ""
}

variable "ec2_instance_profile" {
  default = "XRayDaemonTestingRole"
}

variable "ssh_key_name" {
  default = ""
}

variable "sshkey_s3_bucket" {
  default = ""
}

variable "sshkey_s3_private_key" {
  default = ""
}

variable "aws_region" {
  default = "us-west-2"
}

variable "debug" {
  type    = bool
  default = false
}

variable "trace_doc_file_path" {
  default = "../../trace_document.txt"
}

variable "trace_doc_file_name" {
  default = "trace_document.txt"
}