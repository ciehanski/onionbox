# onionbox

A basic implementation of [OnionShare](https://github.com/micahflee/onionshare) in Go.
Mostly built as a fun project, onionbox is still a WIP so usage is not guaranteed secure, *yet*.

#### Key Features:
- All files are stored in memory and *never* written to disk. The bytes from
each uploaded file are written to an individual **zip buffer** and then written directly
to the response for download. Zip was chosen since it is the most universal archiving
standard that is supported by all operating systems.
- You have the ability to encrypt the uploaded files' bytes if
the content is extra sensitive. GCM is used for encryption. This means, while stored in memory, the files' bytes
will be encrypted as well. Recipients will need to enter the correct password to download or the
presented zip archive will be corrupt.
- You have the ability to limit the number of downloads per download link
generated.
- You have the ability to have download links automatically expire after a duration of your choosing.
- Universal file-sharing. For instance, if you are the recipient of confidential information 
but the sender is not technically-savvy, you yourself can run an onionbox server, send them the 
generated .onion URL and have them upload the files directly for you to download.
- Can be run in a Docker container, or locally on your host machine. You could
of course deploy onionbox to any cloud provider of your choosing.
- Lean as hell. I aim to only rely on the standard library provided by Go to make this software. 
The less dependencies, the better.
- Static binary! Woo! Possible ARM support.

## TODO:
- [ ] Implement tests
- [x] Use flags for config options
- [x] Serve files from buffer instead of disk
- [x] Implement download limits  
- [x] Implement password protected files
- [ ] Implement my own name generator to remove dependency on [randomdata](https://github.com/Pallinder/go-randomdata).
All other dependencies are required to interface with Tor.
- [x] Static build
- [x] Docker build
- [ ] Get docker-compose working with a dnscrypt-proxy. Maybe overkill or moot
but sounds cool as hell, right?
- [ ] ARM support?

## License:
- MIT