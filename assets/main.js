var terminal

function init(shellPath) {
  terminal = new Terminal({
    screenKeys: true,
    useStyle: true,
    cursorBlink: true,
    fullscreenWin: true,
    maximizeWin: true,
    screenReaderMode: true,
    cols: 128,
    fontFamily: 'Terminal, monospace'
  })


  var protocol = (location.protocol === "https:") ? "wss://" : "ws://"
  var url = protocol + location.host + shellPath
  var ws = new WebSocket(url)
  var attachAddon = new AttachAddon.AttachAddon(ws)
  var fitAddon = new FitAddon.FitAddon()

  terminal.loadAddon(attachAddon)
  terminal.loadAddon(fitAddon)
  terminal.open(document.getElementById("terminal"))
  terminal._initialized = true

    function ping() {
      if (ws && ws.readyState === 1) {
          var msg = new TextEncoder().encode("\x01PING" )
          ws.send(msg);
      }
      setTimeout(ping, 1000);
    }

  ws.onclose = function(event) {
    terminal.write('\r\n\nconnection has been closed\n')
  }

  ws.onopen = function() {

    terminal.focus()
    setTimeout(function() {fitAddon.fit()})

    terminal.onResize(function(event) {
      var rows = event.rows
      var cols = event.cols
      var resize = JSON.stringify({cols: cols, rows: rows + 1})
      var msg = new TextEncoder().encode("\x01" + resize)
      ws.send(msg)
    })

    ping()

    window.onresize = function() {
      fitAddon.fit()
    }
  }
}


