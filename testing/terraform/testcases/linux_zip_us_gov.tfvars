testing_ami = "amazonlinux2"
daemon_file_name = "aws-xray-daemon-linux-3.x.zip"
daemon_install_command = "unzip aws-xray-daemon-linux-3.x.zip -d ./xray_daemon"
daemon_start_command = "nohup sudo ./xray_daemon/xray --log-level debug --log-file ./xray_daemon/xray_log &"
daemon_package_local_path = "../../distributions/aws-xray-daemon-linux-3.x.zip"