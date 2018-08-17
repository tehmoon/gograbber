### Gograbber

Gograbber is a tool that takes as input a stream of URL paths and use the print-to-pdf feature of `Chrome`.

It is meant to be used as part of your recon with `gobuster` like this:

```
gobuster -u <target url> -w <wordlist> | tee >(./gograbber -u <same url as gobuster> -d .)
```

You can also set a proxy if you need to do some `TLS` or `burp` rules.

### Building

  - install [Go](https://golang.org)
  - `mkdir ~/.go; export GOPATH=~/.go`
  - `git clone https://github.com/tehmoon/gograbber`
  - `cd gograbber`
  - `go get ./...`
  - `go build .`
  - `./gograbber`

### CAVEATS

Not a finished project at all, for now it spits everything in a directory where `/` are replaced with `_`.

Works only on `Linux`.

### TODO:

  - Add proxy to capture()
  - Refact capture flow()
  - Filename include host
  - Include docker instructions
  - Add user-data-dir flag if you want to reuse
  - Create new user-data-dir everytime
  - Drop privs if ran by root

### Contribution

PR/issues/ideas are welcome.
