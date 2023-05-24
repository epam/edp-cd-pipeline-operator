FROM alpine:3.18.0

ENV OPERATOR=/usr/local/bin/cd-pipeline-operator \
    USER_UID=1001 \
    USER_NAME=cd-pipeline-operator \
    HOME=/home/cd-pipeline-operator

# install operator binary
COPY ./dist/go-binary ${OPERATOR}

COPY build/bin /usr/local/bin
COPY build/pipelines /usr/local/bin/pipelines

RUN  chmod u+x /usr/local/bin/user_setup && chmod ugo+x /usr/local/bin/entrypoint && /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
