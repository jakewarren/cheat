# cheat
[![Build Status](https://github.com/jakewarren/cheat/workflows/lint/badge.svg)](https://github.com/jakewarren/cheat/actions)
[![GitHub release](http://img.shields.io/github/release/jakewarren/cheat.svg?style=flat-square)](https://github.com/jakewarren/cheat/releases])
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](https://github.com/jakewarren/cheat/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/jakewarren/cheat)](https://goreportcard.com/report/github.com/jakewarren/cheat)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=shields)](http://makeapullrequest.com)
> cheat allows you to create, view, and edit cheatsheets on the command-line.

This project is a re-implementation of [Chris Lane's cheatsheet](https://github.com/chrisallenlane/cheat) script in Go along with a few added features and some opinionated formatting & colorization. 

![](screenshot.jpg)

## Install
### Option 1: Binary

Download the latest release from [https://github.com/jakewarren/cheat/releases/latest](https://github.com/jakewarren/cheat/releases/latest)

### Option 2: From source

```
go get github.com/jakewarren/cheat
```

## Usage

### Configuring
`cheat` is configured by a JSON file located at `~/.cheatrc`. A default config will be generated for you:

```json
{
    "highlight": true,
    "cheatdirs": [
        "~/.cheat"
    ],
    "editor": "vim"
}
```

| Configuration Option | Description                                                                                                                                 |
|----------------------|---------------------------------------------------------------------------------------------------------------------------------------------|
| highlight            | controls the colorization of the output                                                                                                     |
| cheatdirs            | an array of directories to search for cheatsheets in. when a new cheatsheet is created, it will be placed in the first directory specified. |
| editor               | specifies the editor to be called when editing a cheat sheet                                                                                |

### Cheatsheets

Cheatsheets are plain-text files with no file extension, and are named according to the command used to view them:

```
cheat tar     # file is named "tar"
cheat git     # file is named "git"
```

Example cheatsheet syntax (left) and display (right):

![](demo.png)

### Bash Completion

To enable bash completion for `cheat` place the following in your `~/.bashrc` file:

```sh
function _cheat_autocomplete {
    sheets=$(cheat -l | cut -d' ' -f1)
    COMPREPLY=()
    # $cur contains the current word the user is entering and is built from:
    #   @COMP_WORDS - array of individual words in the current command line
    #   $COMP_CWORD - index variable into @COMP_WORDS of the word containing the current cursor position
    cur="${COMP_WORDS[COMP_CWORD]}"
    # $prev contains the previous word the user entered
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    if [ $COMP_CWORD = 1 ]; then
        COMPREPLY=(`compgen -W "$sheets" -- $2`)
    fi

        #provide completion for the second word
    if [[ $COMP_CWORD == 2 && $prev == "-e" ]] ; then
        COMPREPLY=(`compgen -W "$sheets" -- $2`)
    fi
}

complete -F _cheat_autocomplete cheat
```

## Acknowledgements
- [cheat/cheat](https://github.com/cheat/cheat)
- [dufferzafar/cheat](https://github.com/dufferzafar/cheat)

## Changes

All notable changes to this project will be documented in the [changelog].

The format is based on [Keep a Changelog](http://keepachangelog.com/) and this project adheres to [Semantic Versioning](http://semver.org/).

## License

MIT © 2019 Jake Warren

[changelog]: https://github.com/jakewarren/cheat/blob/master/CHANGELOG.md
