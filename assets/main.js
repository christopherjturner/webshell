let terminal
let ws
let attachAddon
const baseFalloff = 1000
const maxFalloff = 30000
let falloff = baseFalloff
let fitAddon
let pingTimer = null
let resizeHandler = null


let terminalConfig = {
        screenKeys: true,
        useStyle: true,
        cursorBlink: true,
        fullscreenWin: true,
        maximizeWin: true,
        screenReaderMode: true,
        fontFamily: 'Terminal, monospace',
        scrollOnUserInput: true
    }

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

function reconnect(url) {

    ws = new WebSocket(url)

    ws.onclose = function () {
        stopPing()
        if (attachAddon) attachAddon.dispose()

        terminal.write('\r\n\nTerminal connection closed\r\n')
        setTimeout(() => reconnect(url), falloff)
        falloff = Math.min(falloff * 2, maxFalloff)
    }

    ws.onopen = function () {
        falloff = baseFalloff
        attachAddon = new AttachAddon.AttachAddon(ws)
        terminal.loadAddon(attachAddon)

        terminal.focus()
        setTimeout(function () {
            fitAddon.fit()
        })

        if (ws.readyState === 1) {
            const msg = new TextEncoder().encode("\x01SIZE " + terminal.cols + " " + (terminal.rows + 1))
            ws.send(msg)
        }

        //startPing()

        window.onresize = debounce(function () {
            fitAddon.fit()
        })
    }

}

function schedulePing() {
    pingTimer = setTimeout(function() {
        try {
            if (ws && ws.readyState === 1) {
                const msg = new TextEncoder().encode("\x01PING")
                ws.send(msg)
            }
        } catch (e) {
            console.error("ping failed")
        }
        schedulePing()
    }, 5000)
}

function startPing() {
    stopPing()
    schedulePing()
}

function stopPing() {
    if (pingTimer !== null) {
        clearTimeout(pingTimer)
        pingTimer = null
    }
}

function init(shellPath) {
    terminal = new Terminal(terminalConfig)

    // make the background match the terminal's background
    if (terminalConfig.theme?.background) {
        document.getElementById('terminal').style.background = terminalConfig.theme.background
    }

    const protocol = (location.protocol === "https:") ? "wss://" : "ws://"
    const url = protocol + location.host + shellPath

    reconnect(url)
    //ws = new WebSocket(url)
    //attachAddon = new AttachAddon.AttachAddon(ws)
    fitAddon = new FitAddon.FitAddon()

    //terminal.loadAddon(attachAddon)
    terminal.loadAddon(fitAddon)
    terminal.open(document.getElementById("terminal"))
    terminal._initialized = true

    resizeHandler = debounce(function (event) {
        if (!ws || ws.readyState !== 1) {
            return
        }
        const rows = event.rows
        const cols = event.cols

        console.log(`resizing col:${cols} row:${rows}`)
        const msg = new TextEncoder().encode("\x01SIZE " + cols + " " + (rows + 1))
        ws.send(msg)
    })
    terminal.onResize(resizeHandler)

    const fileTab = document.getElementById('tab-2')
    if(fileTab) {
        fileTab.addEventListener('change', reloadFiles)
    }
}
