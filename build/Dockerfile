FROM alpine:3.8

ENV OPERATOR=/usr/local/bin/cd-pipeline-handler-controller \
    USER_UID=1001 \
    USER_NAME=cd-pipeline-handler-controller
    HOME=/home/cd-pipeline-handler-controller \

# install operator binary
COPY build/_output/bin/cd-pipeline-handler-controller ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
