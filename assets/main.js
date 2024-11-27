let terminal
let ws

function debounce(func, timeout = 300) {
    let timer;
    return (...args) => {
        clearTimeout(timer)
        timer = setTimeout(() => func.apply(this, args), timeout)
    }
}

function reloadFiles() {
    const frame = document.getElementById("file-frame")
    frame.src = frame.src
}


var fitAddon
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
    ws = new WebSocket(url)
    const attachAddon = new AttachAddon.AttachAddon(ws)
    fitAddon = new FitAddon.FitAddon()

    const terminalDiv = document.getElementById("terminal")
    terminal.loadAddon(attachAddon)
    terminal.loadAddon(fitAddon)
    terminal.open(terminalDiv)
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
        terminal.write('\r\n\nTerminal connection closed\r\n')
    }

    ws.onopen = function () {
        terminal.focus()
        setTimeout(resizeTerm)

        terminal.onResize(function (event) {
            const rows = event.rows -1
            const cols = event.cols

            console.log(`resizing col:${cols} row:${rows}`)
            const msg = new TextEncoder().encode("\x01SIZE " + cols + " " + rows)
            ws.send(msg)
        })

        ping()
    }


    function resizeTerm() {
        requestAnimationFrame(() => {
            const height = window.innerHeight;
            document.body.style.height = height + 'px'
        })
        requestAnimationFrame(() => {
            fitAddon.fit()
            console.log("window resized " + window.innerHeight + " div " +  document.getElementById("terminal").style.height)
        });
    }

    window.onresize = debounce(resizeTerm)

    const fileTab = document.getElementById('tab-2')
    if(fileTab) {
        fileTab.addEventListener('change', reloadFiles)
    }
}

