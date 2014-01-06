FROM ubuntu

ADD logger /usr/local/bin/
ENTRYPOINT ["logger"]
CMD ["-s", "irc.freenode.net:6667", "-nick", "dockerstats", "#docker", "#docker-dev"]
