FROM debian

COPY cmd/cmd /usr/bin/cdpshell

CMD [ /usr/bin/cdpshell ]
