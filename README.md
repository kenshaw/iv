# iv

`iv` is a command-line image viewer using terminal graphics (Sixel, iTerm,
Kitty).

<p align="center">
  <a href="#installing" title="Installing">Installing</a> |
  <a href="#building" title="Building">Building</a> |
  <a href="#using" title="Using">Using</a> |
  <a href="https://github.com/kenshaw/iv/releases" title="Releases">Releases</a>
</p>

[![Releases][release-status]][Releases]
[![Discord Discussion][discord-status]][discord]

[releases]: https://github.com/kenshaw/iv/releases "Releases"
[release-status]: https://img.shields.io/github/v/release/kenshaw/iv?display_name=tag&sort=semver "Latest Release"
[discord]: https://discord.gg/WDWAgXwJqN "Discord Discussion"
[discord-status]: https://img.shields.io/discord/829150509658013727.svg?label=Discord&logo=Discord&colorB=7289da&style=flat-square "Discord Discussion"

## Overview

`iv` is a command-line image viewer using terminal graphics.

Uses [Sixel][sixel], [iTerm Inline Images][iterm], or [Kitty][kitty] graphics
protocols where available. See [Are We Sixel Yet?][arewesixelyet] for a list of
terminals known to work with this package.

[sixel]: https://saitoha.github.io/libsixel/
[iterm]: https://iterm2.com/documentation-images.html
[kitty]: https://sw.kovidgoyal.net/kitty/graphics-protocol/
[arewesixelyet]: https://www.arewesixelyet.com

## Installing

`iv` can be installed [via Release][], [via Homebrew][], [via AUR][], [via
Scoop][] or [via Go][]:

[via Release]: #installing-via-release
[via Homebrew]: #installing-via-homebrew-macos-and-linux
[via AUR]: #installing-via-aur-arch-linux
[via Scoop]: #installing-via-scoop-windows
[via Go]: #installing-via-go

### Installing via Release

1. [Download a release for your platform][releases]
2. Extract the `iv` or `iv.exe` file from the `.tar.bz2` or `.zip` file
3. Move the extracted executable to somewhere on your `$PATH` (Linux/macOS) or
   `%PATH%` (Windows)

### Installing via Homebrew (macOS and Linux)

Install `iv` from the [`kenshaw/iv` tap][iv-tap] in the usual way with the [`brew`
command][homebrew]:

```sh
# install
$ brew install kenshaw/iv/iv
```

### Installing via AUR (Arch Linux)

Install `iv` from the [Arch Linux AUR][aur] in the usual way with the [`yay`
command][yay]:

```sh
# install
$ yay -S iv-cli
```

Alternately, build and [install using `makepkg`][arch-makepkg]:

```sh
# clone package repo and make/install package
$ git clone https://aur.archlinux.org/iv-cli.git && cd iv-cli
$ makepkg -si
==> Making package: iv-cli 0.4.4-1 (Sat 11 Nov 2023 02:28:28 PM WIB)
==> Checking runtime dependencies...
==> Checking buildtime dependencies...
==> Retrieving sources...
...
```

### Installing via Scoop (Windows)

Install `iv` using [Scoop](https://scoop.sh):

```powershell
# Optional: Needed to run a remote script the first time
> Set-ExecutionPolicy RemoteSigned -Scope CurrentUser

# install scoop if not already installed
> irm get.scoop.sh | iex

# install iv with scoop
> scoop install iv
```

### Installing via Go

Install `iv` in the usual Go fashion:

```sh
# install latest iv version
$ go install github.com/kenshaw/iv@latest
```

## Using

```sh
$ iv /path/to/image_or_directory

# all command line options
$ iv --help
```

[homebrew]: https://brew.sh/
[iv-tap]: https://github.com/kenshaw/homebrew-iv
[aur]: https://aur.archlinux.org/packages/iv-cli
[arch-makepkg]: https://wiki.archlinux.org/title/makepkg
[yay]: https://github.com/Jguer/yay
