Name:		xray
Version:	%rpmversion
Release:	1
Summary:	X-Ray daemon

Group:		Amazon/Tools
License:	Apache License, Version 2.0
URL:		http://docs.aws.amazon.com/

%description
This package provides daemon to send trace segments to xray dataplane

%files
/usr/bin/xray

%config(noreplace) /etc/amazon/xray/cfg.yaml
%config(noreplace) /etc/init/xray.conf
%config(noreplace) /etc/systemd/system/xray.service

%pre
# First time install create user and folders
if [ $1 -eq 1 ]; then
    useradd --system -s /bin/false xray
    mkdir -m 755 /var/log/xray/
    chown xray /var/log/xray/
else
# Stop the agent before the upgrade
    if [ $1 -ge 2 ]; then
        if [[ `/sbin/init --version` =~ upstart ]]; then
            /sbin/stop xray
        elif [[ `systemctl` =~ -\.mount ]]; then
            systemctl stop xray
            systemctl daemon-reload
        fi
    fi
fi

%preun
# Stop the agent after uninstall
if [ $1 -eq 0 ] ; then
    if [[ `/sbin/init --version` =~ upstart ]]; then
        /sbin/stop xray
        sleep 1
    elif [[ `systemctl` =~ -\.mount ]]; then
        systemctl stop xray
        systemctl daemon-reload
    fi
fi

%posttrans
# Start the agent after initial install or upgrade
if [ $1 -ge 0 ]; then
    if [[ `/sbin/init --version` =~ upstart ]]; then
        /sbin/start xray
    elif [[ `systemctl` =~ -\.mount ]]; then
        systemctl start xray
        systemctl daemon-reload
    fi
fi

%clean
# rpmbuild deletes $buildroot after building, specifying clean section to make
# sure it is not deleted

