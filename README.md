# Waiter

A game server for Cube 2: Sauerbraten, written in Go.


## To Do

Next step is to implement more network events to support effic mode completely, then efficctf, then insta and instactf. After that, faf, then capture and regen capture would be the next goals.


## Project Structure

Most functionality is organized into internal packages. [`/cmd/waiter/`](/cmd/waiter/) contains the actual command to start a server, i.e. configuration file parsing, initialization of all components, and handling of incoming packets. Protocol definitions like network message codes can be found in [`internal/definitions`](/internal/definitions/).

Other interesting packages:

- [`cubecode`](cubecode)
- [`internal/auth`](internal/auth)
- [`internal/protocol/enet`](internal/protocol/enet)
- [`internal/definitions`](internal/definitions)
- [`internal/masterserver`](internal/masterserver)

In [`/cmd/genauth/`](/cmd/genauth/), there is a command to generate auth keys for users. This server uses a different representation for public keys, so the output of `/genauthkey` in the vanilla client will be useless.


## License

This code is licensed under a BSD License:

Copyright (c) 2014-2017 Alexander Willing. All rights reserved.

Redistribution and use in source and binary forms, with or without modification,
are permitted provided that the following conditions are met:

- Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
- Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
