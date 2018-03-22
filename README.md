# git-cgi-server

Simple Git CGI Server (using git-http-backend) written in Go

## Requires

* Git (including git-http-backend)

## Features

* Simple and lightweight
* Support HTTP authentication (Basic and Digest)

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

See `git-cgi-server -h` for more options.

## License

Apache 2.0 License

## Author

Yuki (a.k.a. pasela)
