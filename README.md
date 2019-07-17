# pacman-smartmirror
An easy to use smart mirror server for pacman that will automatically download packats on the fly when requested,
cache them and auto-update cached packets. Perfect for LAN usage on an inexpensive Raspberry Pi.

## Installation
### Server
#### [Gobin](https://github.com/myitcv/gobin)
`gobin github.com/veecue/pacman-smartmirror`
### Client
Add `Server = http://hostname:41234/$repo/os/$arch` at the beginning of `/etc/pacman.d/mirrorlist`
If you are using another port than the default 41234, use that one

## Usage
### Client
When the mirrorlist is configured as in [Installation](#installation), use `pacman` just as usual.
### Server
```
Usage of pacman-smartmirror:
  -d string
        Existing directory to use for the cached packages
  -l string
        Address and port for the HTTP server to listen on (default ":41234")
  -m string
        Filename of the mirrorlist to use (use /etc/pacman.d/mirrorlist on arch)

```
