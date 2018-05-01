FROM scratch
MAINTAINER Brian Hechinger <wonko@4amlunch.net>

ADD role-cmd-linux-amd64 role-cmd
VOLUME /etc/chremoas

ENTRYPOINT ["/role-cmd", "--configuration_file", "/etc/chremoas/chremoas.yaml"]
