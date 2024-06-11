FROM debian:bookworm-slim

# Redis CLI and misc os stuff
RUN apt update && \
    apt install -y curl wget unzip redis-tools less && \
    apt clean && \
    rm -rf /var/lib/apt/lists/*

# AWS CLI
RUN curl -s -o aws.zip  "https://awscli.amazonaws.com/awscli-exe-linux-$(uname -m).zip" && \
    unzip aws.zip && \
    ./aws/install && \
    rm -rf ./aws && \
    rm aws.zip

# Mongo Shell
RUN curl -s -o mongosh.deb "https://downloads.mongodb.com/compass/mongodb-mongosh_2.2.6_amd64.deb" && \
    dpkg -i mongosh.deb && \
    rm mongosh.deb

# Mongo Tools
RUN curl -s -o mongotools.deb "https://fastdl.mongodb.org/tools/db/mongodb-database-tools-debian12-x86_64-100.9.4.deb" && \
    dpkg -i mongotools.deb && \
    rm mongotools.deb

COPY cdpshell /usr/bin/cdpshell

RUN useradd -ms /bin/bash cdpshell
USER cdpshell
WORKDIR /home/cdpshell

ENTRYPOINT [ "/usr/bin/cdpshell", "-home", "/home/cdpshell" ]
