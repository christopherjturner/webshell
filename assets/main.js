var terminal

class XtermWebfont {
  activate(terminal) {
    this._terminal = terminal;
    terminal.loadWebfontAndOpen = function (element) {
      const fontFamily = this.options.fontFamily;
      const regular = new FontFaceObserver(fontFamily).load();
      const bold = new FontFaceObserver(fontFamily, {
        weight: "bold"
      }).load();
      return regular.constructor.all([regular, bold]).then(() => {
        this.open(element);
        return this;
      }, () => {
        this.options.fontFamily = "Courier";
        this.open(element);
        return this;
      });
    };
  }
  dispose() {
    delete this._terminal.loadWebfontAndOpen;
  }
};

function init(shellPath) {
  terminal = new Terminal({
    screenKeys: true,
    useStyle: true,
    cursorBlink: true,
    fullscreenWin: true,
    maximizeWin: true,
    screenReaderMode: true,
    cols: 128,
    fontFamily: 'DejaVu Sans Mono'
  });



  var protocol = (location.protocol === "https:") ? "wss://" : "ws://";
  var url = protocol + location.host + shellPath
  var ws = new WebSocket(url);
  var attachAddon = new AttachAddon.AttachAddon(ws);
  var fitAddon = new FitAddon.FitAddon();

  terminal.loadAddon(attachAddon);
  terminal.loadAddon(fitAddon);
  //terminal.loadAddon(new XtermWebfont()) // TODO: figure out how to get this to actually load webfonts!
  terminal._initialized = true;

  terminal.open(document.getElementById("terminal"));
  //terminal.loadWebfontAndOpen(document.getElementById("terminal"));


  ws.onclose = function(event) {
    console.log(event);
    terminal.write('\r\n\nconnection has been closed\n')
  };

  ws.onopen = function() {

    terminal.focus();
    setTimeout(function() {fitAddon.fit()});

    terminal.onResize(function(event) {
        console.log("resizing")
      var rows = event.rows;
      var cols = event.cols;
      var size = JSON.stringify({cols: cols, rows: rows + 1});
      var send = new TextEncoder().encode("\x01" + size);
      ws.send(send);
    });

    window.onresize = function() {
      fitAddon.fit();
    };
  };
};

function refreshFiles() {
  const div = document.getElementById("files")
  const http = new XMLHttpRequest()
  http.onload = function() {
    div.innerHTML = http.responseText
  }
  http.open("GET", "./home")
  http.send()
}

function initFilePage() {
// file upload handler
document.getElementById('uploadForm').addEventListener('submit', function(event) {
    event.preventDefault()

    const fileInput = document.getElementById('fileInput');
    const file = fileInput.files[0];

    if (!file) {
        return;
    }

    const formData = new FormData();
    formData.append('file', file);

    fetch('./upload', {
        method: 'POST',
        body: formData,
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Network response was not ok ' + response.statusText);
        }
        refreshFiles()
        fileInput.value = ''
        return
    })
    .catch(error => {
        console.error('Error:', error);
        fileInput.value = ''
        alert('File upload failed.');
    });
});
}
