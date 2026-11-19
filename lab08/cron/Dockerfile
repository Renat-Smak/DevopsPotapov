FROM alpine:3.23
RUN command -v crond && mkdir -p /data
COPY crontab /etc/crontabs/root
COPY record-time.sh /usr/local/bin/record-time
RUN chmod 0600 /etc/crontabs/root && chmod 0755 /usr/local/bin/record-time
ENTRYPOINT ["crond", "-f", "-l", "5", "-L", "/dev/stdout"]
