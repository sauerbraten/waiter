# Waiter

A game server for [Cube 2: Sauerbraten](http://sauerbraten.org/).

    /connect p1x.pw


## Features

What works:

- connecting & disconnecting
- relaying positional data between clients (at 91 Hz!)
- chat, team chat
- changing weapon, shooting, killing, suiciding, spawning (but rocket and grenade don't show up for other players)
- global auth
- local auth
- changing mastermode
- forcing gamemode and/or map
- pausing & resuming
- changing your name
- extinfo (server mod ID: -9)

Pretty much everything else is not yet implemented:

- spawning pick ups (ammo, armour, quad, flags, ...)
- picking up those pick ups
- any modes besides DM (only insta is fully playable right now)
- demo recording
- `/checkmaps` (will compare against server-side hash, not majority)
- overtime (& maybe golden goal)

Some things are specifically not planned and will likely never be implemented:

- bots
- map voting
- coop edit mode (including `/sendmap` and `/getmap`)
- claiming privileges using `/setmaster 1` (relinquishing them with `/setmaster 0` and sharing master using `/setmaster 1 <cn>` already works)


## Installing

Make sure you have Go installed as well as the ENet development headers (on Fedora, `sudo dnf install enet-devel`). Run `go install github.com/sauerbraten/waiter/cmd/waiter` to install the `waiter` command in your `$GOPATH/bin` (which should by in your `$PATH`).

The server requires `config.json`, `bans.json` and `users.json` to be placed in the working directory.


## Building

Make sure you have Go installed as well as the ENet development headers (on Fedora, `sudo dnf install enet-devel`). Clone the repository, `cd waiter/cmd/waiter`, then `go build`.

You can then start the server with `./waiter`.


## To Do

Figure out why projectiles (grenades and rockets) are not rendered for players other than the one shooting.

Then, implement mode/map change (forced, voting maybe later). With that, the goal is to support insta and effic games completely so waiter can be used for duels at least.

Future goals will be efficctf and instactf (flag spawns & events), then ffa (all the other items), then capture and regen capture (capture base events).


## Project Structure

Most functionality is organized into internal packages. [`/cmd/waiter/`](/cmd/waiter/) contains the actual command to start a server, i.e. configuration file parsing, initialization of all components, and handling of incoming packets. Protocol definitions like network message codes can be found in [`internal/definitions`](/internal/definitions/).

Other interesting packages:

- [`pkg/protocol`](pkg/protocol) & [`pkg/protocol/cubecode`](pkg/protocol/cubecode)
- [`internal/auth`](internal/auth)
- [`internal/net/enet`](internal/net/enet)
- [`internal/masterserver`](internal/masterserver)

In [`cmd/genauth`](cmd/genauth), there is a command to generate auth keys for users. This server uses a different representation for public keys, so the output of `/genauthkey` in the vanilla client will be useless.


## Why?

I started this mainly as a challenge to myself and because I have ideas to improve the integration of Sauerbraten servers with other services and interfaces. For example, making the server state and game events available via WebSockets in real-time, instead of the UDP-based extinfo protocol, and integrating a third-party auth system (spanning multiple servers).

Writing a server that makes it easy to modify gameplay is not one of the goals of this project, neither is plugin support, although it might happen at some point. If you want that, now, use pisto's great [spaghettimod](https://github.com/pisto/spaghettimod).


## License

This code is licensed under a BSD License:

Copyright (c) 2014-2019 Alexander Willing. All rights reserved.

Redistribution and use in source and binary forms, with or without modification,
are permitted provided that the following conditions are met:

- Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
- Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
