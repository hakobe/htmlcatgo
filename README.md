# htmlcatgo

golang clone of htmlcat https://github.com/motemen/App-htmlcat - stdin to your browser

## Feature

`htmlcatgo` brings stdin input to html with creating new HTTP server.

This requires your browser to support Server-Sent Event.

## Install

```sh
$ go get github.com/hakobe/htmlcatgo
$ go install github.com/hakobe/htmlcatgo
```

## Usage

```sh
$ tail -F /var/log/message | htmlcatgo
```

You will see message below

```sh
2013/12/16 08:15:02 htmlcatgo: http://localhost:45273
```

and you can see output of `tail -F /var/log/message` at http://localhost:45273 on your browser.

## Author

Yohei Fushii <hakobe@gmail.com>
