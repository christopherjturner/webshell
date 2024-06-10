FROM mongo:6

RUN apt update && apt install -y curl unzip redis-tools less && apt clean &&  rm -rf /var/lib/apt/lists/*
RUN curl -s -o aws.zip "https://awscli.amazonaws.com/awscli-exe-linux-$(uname -m).zip" && unzip aws.zip && ./aws/install && rm -rf ./aws && rm aws.zip

COPY cdpshell /usr/bin/cdpshell

RUN useradd -ms /bin/bash cdpshell
USER cdpshell
WORKDIR /home/cdpshell

ENTRYPOINT [ "/usr/bin/cdpshell", "-home", "/home/cdpshell" ]
