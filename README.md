# Web shell

## What is it?
Webshell is xterm.js based terminal for remote access to systems with additional auditing and authentication options.

## Start the web shell
- git clone https://github.com/christopherjturner/webshell
- install go https://go.dev/doc/install

```bash
go run . -port 8085 -token 12345
```

## Open the web shell
http://localhost:8085/12345/

## Open the web shell via the CDP proxy
http://localhost:8000/12345/

## Themes

You can customize the terminal theme using the `-theme theme-file.js` option.
A theme file is simply a javascript file that sets the values of `terminal.theme`.
Its possible to set other xterm.js options from here.

```js
terminal.theme = {
    foreground: '#708284',
    background: '#001e27',
    cursor: '#708284',
    black: '#002831',
    brightBlack: '#001e27',
    red: '#d11c24',
    brightRed: '#bd3613',
    green: '#738a05',
    brightGreen: '#475b62',
    yellow: '#a57706',
    brightYellow: '#536870',
    blue: '#2176c7',
    brightBlue: '#708284',
    magenta: '#c61c6f',
    brightMagenta: '#5956ba',
    cyan: '#259286',
    brightCyan: '#819090',
    white: '#eae3cb',
    brightWhite: '#fcf4dc'
}
```

