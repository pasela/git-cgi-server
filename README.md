# git-cgi-server

Simple Git Smart HTTP Server (using git-http-backend) written in Go.

## What is Git Smart HTTP?

See: https://git-scm.com/book/en/v2/Git-on-the-Server-Smart-HTTP

## Features

* Simple and lightweight
* Support HTTP authentication (Basic and Digest)
* Support TLS

## Motivation

* I don't want to open SSH only for Git access.
* I don't want to use rich HTTP servers (such as Apache, nginx, etc.) only for Git.
* I don't want to change owner of repositories to the httpd user (such as httpd, www-data, etc.)
* I want to launch easily without any complex installation and configuration.

## Requirement

* Git (including git-http-backend)

## Install

```sh
go get github.com/pasela/git-cgi-server
```

## Usage

```sh
git-cgi-server [OPTIONS] [REPOS_DIR]
```

Export all repositories:
```sh
git-cgi-server --export-all /path/to/repos
```

Enable Basic authentication:
```sh
git-cgi-server --basic-auth-file=/path/to/.htpasswd --auth-realm=MyGitRepos /path/to/repos
```

Use TLS:
```sh
git-cgi-server --cert-file=/path/to/server.crt --key-file=/path/to/server.key /path/to/repos
```

See `git-cgi-server -h` for more options.

## License

Apache 2.0 License

## Author

Yuki (a.k.a. pasela)
