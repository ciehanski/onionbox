# onionbox [![Build Status](https://travis-ci.com/ciehanski/onionbox.svg?branch=master)](https://travis-ci.com/ciehanski/onionbox) [![Go Report Card](https://goreportcard.com/badge/github.com/ciehanski/onionbox)](https://goreportcard.com/report/github.com/ciehanski/onionbox) [![codecov](https://codecov.io/gh/ciehanski/onionbox/branch/master/graph/badge.svg)](https://codecov.io/gh/ciehanski/onionbox)

A basic implementation of [OnionShare](https://github.com/micahflee/onionshare) in Go.
Mostly built as a fun project, onionbox is still a WIP so usage is not guaranteed secure, *yet*.

#### Key Features:
- All files are stored in memory and *never* written to disk. The bytes from
each uploaded file are written to an individual **zip buffer** (in memory, and also compressed 😄) and then written directly
to the response for download. Zip was chosen since it is the most universal archiving
standard that is supported by all operating systems.
- You have the ability to encrypt the uploaded files' bytes if
the content is extra sensitive. GCM is used for encryption. This means, while stored in memory, the files' bytes
will be encrypted as well. **If password encryption is enabled, recipients will need to enter the correct password 
before the download.**
- You have the ability to limit the number of downloads per download link
generated.
- You have the ability to enforce that download links automatically expire after a specific duration of your choosing.
- 2-way file sharing. For instance, if you are the recipient of confidential information 
but the sender is not technically-savvy, you yourself can run an onionbox server, send them the 
generated .onion URL and have them upload the files directly for you to download.
- Can be run in a Docker container, or locally on your host machine. You could
of course deploy onionbox to any cloud provider of your choosing.
- Static binary! Woo!

## TODO:
- [ ] Implement more tests
- [ ] Implement usage of *multipart.Reader
- [ ] Android support (build almost working)
- [ ] Windows support (needs testing)
- [x] ARM support
- [ ] Get a docker-compose config working with a dnscrypt-proxy. Maybe overkill or moot
but sounds cool as hell, right?

## Contributing:

Any pull request submitted must meet the following requirements:
- Have included tests applicable to the relevant PR.
- Attempt to adhere to the standard library as much as possible.

You can get started by either forking or cloning the repo. After, you can get started
by running:

```bash
make run
```

This will go ahead and build everything necessary to interface with Tor. After compose
has completed building, you will have a new `onionbox` container which will be your
dev environment.

Anytime a change to a .go or .mod file is detected the container will rerun with
the changes you have made. You must save in your IDE or text editor for the 
changes to be picked up. It takes roughly ~35 seconds for onionbox to restart after 
you have made changes.

You can completely restart the build with:
```bash
make restart
```

Run tests:
```bash
make exec
make test
```

Get container logs:
```bash
make logs
```

Shell into docker container:
```bash
make exec
```

Lint the project:
```bash
make lint
```

## Shoutouts:
Huge shoutout to [@karalabe](https://github.com/karalabe), the creator of [go-libtor](https://github.com/ipsn/go-libtor) which enables the 
creation of a Go-friendly static Tor lib which utilizes [bine](https://github.com/cretz/bine) (created by [@cretz](https://github.com/cretz))
to interface with the Tor API. Big thanks to these guys or this project would not be possible.

## License:
- MIT