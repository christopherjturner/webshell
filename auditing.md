# Auditing

Auditing consists of two parts:

1. Exec auditing. All commands run by the user is logged.
2. TTY auding. Everything the user sees in the terminal is recorded.

## Exec Auditing

This is implmented using `strace`.
When a user connects to the webshell, webshell launches `/bin/bash` and pipes the down the websocket.
After bash is launched, strace is run attaching to the PID of the bash instance.
Stderr from strace is sent to a reader that filters out only `execve` calls and writes the output to the audit logger.

If strace is not installed then exec level auditing will be disabled.
Going forward we might want to look at implementing the syscall based auditing in pure go using ptrace.


## TTY Recording

This keeps a copy of every byte sent to the user's xterm.js terminal along with a timeline of when this data was sent.
At the end of the users session these files are uploaded to S3.

The TTY Recordings can be later played back via the webshell (see: /replay endpoint).

TTY Recording creates two temporary files while the user's session is in progress.

- `ttyrec.data` the raw output of tty session
- `ttyrec.time` the times at which the tty writes happened

At the end of the session these files are merged into a single file.

### TTY Recording (ttyrec) File Format

The ttyrec has three parts:

1. Header      - Fixed size with data about how to access the rest of the file
2. Audit Data  - Holds the raw tty output
3. Timing Data - Hold the data about output was written to the terminal

### Header

The header is a fixed sized binary structure.
It consists of the following fields:

Magic        4 byte (uint32) - always set to 0xDC3443CD
Version      1 byte          - always 1 (uint32)
Compression  2 byte          - what compression is used. 0=None, 1=gzip. first byte is audit, 2nd timing data
Flags        1 byte          - bitfield of flags. (unused)
AuditOfset   8 byte (int64)  - offset of audit data from the start of the file.
AuditLength  8 byte (int64)  - length of audit data section in bytes
TimingOffset 8 byte (int64)  - offset of audit data from the start of the file.
TimingSize   8 byte (int64)  - length of the timing section in bytes.

### Audit Data
Audit data is the raw TTY output. Its copied from the pseudo-terminal at the same point its written to the websocket. The raw data can be replayed by sending it down the websocket to an attached xterm.js

### Timing Data

Timing data is used to play back the TTY output at roughly the same speed it was displayed.

Time        8 bytes (int64) - Unix time in milliseconds
Offset      8 bytes (int64) - Offset position in the audit data that the event occured

To keep the size of the timing data down it is updated no more than once every 100ms.

### Future Work
Add an extra section to annoate timings with data from the Exec Audit.
Some sort of checksum/signing?

