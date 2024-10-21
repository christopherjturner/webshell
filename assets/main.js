let terminal

function debounce(func, timeout = 300) {
    let timer;
    return (...args) => {
        clearTimeout(timer)
        timer = setTimeout(() => func.apply(this, args), timeout)
    }
}

function init(shellPath) {
    terminal = new Terminal({
        screenKeys: true,
        useStyle: true,
        cursorBlink: true,
        fullscreenWin: true,
        maximizeWin: true,
        screenReaderMode: true,
        fontFamily: 'Terminal, monospace',
        scrollOnUserInput: false
    })


    const protocol = (location.protocol === "https:") ? "wss://" : "ws://"
    const url = protocol + location.host + shellPath
    const ws = new WebSocket(url)
    const attachAddon = new AttachAddon.AttachAddon(ws)
    const fitAddon = new FitAddon.FitAddon()

    terminal.loadAddon(attachAddon)
    terminal.loadAddon(fitAddon)
    terminal.open(document.getElementById("terminal"))
    terminal._initialized = true

    function ping() {
        try {
            if (ws && ws.readyState === 1) {
                const msg = new TextEncoder().encode("\x01PING")
                ws.send(msg);
            }
        } catch (e) {
            console.error("ping failed")
        }
        setTimeout(ping, 5000);
    }

    ws.onclose = function () {
        terminal.write('\r\n\nconnection has been closed\n')
    }

    ws.onopen = function () {
        terminal.focus()
        setTimeout(function () {
            fitAddon.fit()
        })

        terminal.onResize(debounce(function (event) {
            const rows = event.rows
            const cols = event.cols
            const resize = JSON.stringify({ cols: cols, rows: rows + 1 })
            console.log(resize)
            const msg = new TextEncoder().encode("\x01" + resize)
            ws.send(msg)
        }))

        ping()

        window.onresize = debounce(function () {
            fitAddon.fit()
        })
    }
}

