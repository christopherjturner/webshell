FROM debian

COPY cmd/cmd /usr/bin/webshell

CMD [ "/usr/bin/cdpshell" ]
