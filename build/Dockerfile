FROM alpine:3.8

ENV OPERATOR=/usr/local/bin/cd-pipeline-handler-controller \
    USER_UID=1001 \
    USER_NAME=cd-pipeline-handler-controller \
    HOME=/home/cd-pipeline-handler-controller

# install operator binary
COPY cd-pipeline-handler-controller ${OPERATOR}

COPY bin /usr/local/bin
COPY pipelines /usr/local/bin/pipelines
RUN  chmod u+x /usr/local/bin/user_setup && chmod ugo+x /usr/local/bin/entrypoint && /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
