# Auditing

Idea:

In the shell handler:
- Create a new append file
- On the TTY -> WS handler, write all bytes down to the file
- Also record the timing. This doesn't have to be super precise
  - timestamp (int64 value) -> offset in file
  - defer the writing of the file

File Format

The tty log is made up of three sections:

- Header
- Audit Data
- Timing Data

## Header

The header describes the layout of the file.
The header should be read fully before paring the rest of the file.
The locations of the audit and timing section are described in the header.
The compression section describes if either of these sections are compressed, and if so with what alogrithm.

Magic       2 byte          - always 0xCE 0x92
Version     4 byte          - always 0 (uint32)
Compression 2 bytes         - 0x00 no compressions 0x01 gzip. byte 1 = audit section byte 2 = timings section
Timestamp   8 byte (int64)  - unix timestamp when log was started
AuditStart  8 byte (int64)  - length of audit data section in bytes
AuditSize   8 byte (int64)  - length of audit data section in bytes
TimingStart 8 byte (int64)  - offset of the timing section
TimingSize  8 byte (int64)  - offset of the timing section

TODO: maybe record the timing precision if we make it configurable?

## Audit Data

The audit data section contains a raw copy of all the audit data sent to the webshell from the terminal.
It is replayed via websockets to an instance of xterm.js to show exactly what the user would have seen.

## Timing Data

Timing data consists of pairs of 16 bytes.

Offset 8 bytes (int64)
Time   8 bytes (int64)

Timing data is precise to 100ms (TBC this may be configurable), meaning a new entry is created at most once every 100ms.
The offset value points at the location in the audit data the event occured.
Offset values are relative to the start of the audit data section (AuditStart + Offset).
If the audit data is compressed then the offset will be a pointer to the uncompressed version of the data.

