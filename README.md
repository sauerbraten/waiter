# Waiter

A game server for [Cube 2: Sauerbraten](http://sauerbraten.org/).

    /connect p1x.pw


## Features

What works:

- insta, insta team, effic, effic team, tactics, tactics team
- insta ctf, effic ctf
- chat, team chat
- changing weapon, shooting, killing, suiciding, spawning
- global auth (`/auth` and `/authkick`)
- local auth (`/sauth`, `/dauth`, `/sauthkick`, `/dauthkick`, auth-on-connect)
- sharing master
- setting mastermode
- forcing gamemode and/or map
- pausing & resuming (with countdown)
- locking teams (`keepteams` server command)
- queueing maps (`queuemap` server command)
- changing your name
- extinfo (server mod ID: -9)

Server commands:

These can be used either as `#cmd bla foo` or `/servcmd cmd bla foo`:

- `keepteams 0|1` (a.k.a. `persist`): set to 1 to disable randomizing teams on map load
- `queuemap [map...]`: check the map queue or enqueue one or more maps
- `competitive 0|1`: in competitive mode, the server waits for all players to load the map before starting the game, and automatically pauses the game when a player leaves or goes to spectating mode

Pretty much everything else is not yet implemented:

- spawning pick ups (ammo, armour, quad, ...)
- picking up those pick ups
- any modes requiring items (e.g. ffa) or bases (capture) or tokens (collect)
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

- implement ffa (item pick ups), capture and regen capture (capture base events)
- intermission stats (depending on mode)
- #stats command
- competitive mode (auto-pause on leave, specs muted for players, ...)
- store frags, deaths, etc. in case a player re-connects


## Project Structure

Most functionality is organized into internal packages. [`/cmd/waiter/`](/cmd/waiter/) contains the actual command to start a server, i.e. configuration file parsing, initialization of all components, and handling of incoming packets. Protocol definitions like network message codes can be found in [`internal/definitions`](/internal/definitions/).

Other interesting packages:

- [`pkg/protocol`](pkg/protocol) & [`pkg/protocol/cubecode`](pkg/protocol/cubecode)
- [`internal/auth`](internal/auth)
- [`internal/net/enet`](internal/net/enet)
- [`internal/masterserver`](internal/masterserver)

In [`cmd/genauth`](cmd/genauth), there is a command to generate auth keys for users. While you can use auth keys generated with Sauerbraten's `/genauthkey` command, `genauth` provides better output (`auth.cfg` line for the player, JSON object for this server's `users.json` file).


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
