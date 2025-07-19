/*
Control sequences are used to do things like move the cursor, change text color, clear the screen, and more. They are the primary way that programs interact with the terminal.

Programs running within terminals only have a single way to communicate with the terminal: writing bytes to the connected file descriptor (typically a pty). In order to differentiate between text to be displayed and commands to be executed, terminals use special syntax known collectively as control sequences.

Due to the historical nature of terminals, control sequences come in a handful of different formats. Most begin with an escape character (0x1B), so control sequences are sometimes referred to as escape codes or escape sequences.

There are eight types of control sequences:

Control Characters: supported
Escape Sequences: supported
CSI Sequences ("Control Sequence Introducer"): supported
OSC Sequences ("Operating System Command"): supported
DCS Sequences ("Device Control Sequence"):supported
SOS Sequences ("Start Of String"): non-supported
PM Sequences ("Privacy Message"): non-supported
APC Sequences ("Application Program Command"): non-supported

Each type of control sequence has a different format and purpose.

Currently, KAI only supports some control sequences. The supported sequences are:
  - Control Characters
  - Escape Sequences
  - CSI Sequences ("Control Sequence Introducer")
  - OSC Sequences ("Operating System Command")
  - DCS Sequences ("Device Control Sequence")
*/
package sequences
