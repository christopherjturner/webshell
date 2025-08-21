function formatRemaining(start, ttl) {
    const now = Date.now()
    const end = start + ttl * 1000 // ttl is in seconds
    let remaining = Math.max(0, Math.floor((end - now) / 1000)) // in seconds
    const hours = String(Math.floor(remaining / 3600)).padStart(2, '0')
    remaining %= 3600
    const minutes = String(Math.floor(remaining / 60)).padStart(2, '0')
    const seconds = String(remaining % 60).padStart(2, '0')

    return `Session ends in ${hours}:${minutes}:${seconds}`
}

function hideIfClosed(el) {

}

function startTimer(el) {
    try {

        if (!el) return
        const start = parseInt(el.dataset.start, 10) // ms timestamp
        const ttl = parseInt(el.dataset.ttl, 10)     // seconds
        if (ttl <= 0) {
           return
        }

        el.textContent = formatRemaining(start, ttl);
        const interval = setInterval(() => {
            if (ws && ws.readyState === ws.OPEN) {
                el.textContent = formatRemaining(start, ttl);
                if (el.textContent.endsWith("00:00:00")) {
                    clearInterval(interval);
                }
            } else {
                el.textContent = ""
            }

        }, 1000)
    } catch (e) {
        console.log(e)
    }
}
