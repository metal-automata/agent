FROM alpine:latest

ENTRYPOINT ["/usr/sbin/agent"]

COPY agent /usr/sbin/agent
RUN chmod +x /usr/sbin/agent
