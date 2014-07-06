[![Stories in Ready](https://badge.waffle.io/project-douglas/c3d-go.png?label=ready&title=Ready)](https://waffle.io/project-douglas/c3d-go)

## Introduction

C3D => Contract Controlled Content Distribution

More coming.

## How to play

To setup, you'll have to grab the project-douglas versions of the go-ethereum and eth-go. As development on those repos continues, we need a stable repo so our APIs don't continuously break. So, do the following:

```
go get -d github.com/project-douglas/eth-go
cd $GOPATH/src/github/project-douglas/eth-go
git checkout pd
go install


go get -d github.com/project-douglas/go-ethereum/ethereum
cd $GOPATH/src/github/project-douglas/go-ethereum/ethereum
git checkout pd
go install
```

That should get you our versions of the repos and install them. If you don't do `git checkout pd`, you will have the ethereum versions (not the project-douglas versions), and everything will be sure to go to shit from there :)

Now, grab c3d-go: `go get github.com/project-douglas/c3d-go`. That will install it.  If you make changes and want to re-install, just hit `go install` in the c3d-go repo. Run it with `c3d-go`, or `$GOPATH/bin/c3d-go` if you must.  The webapp is at `http://localhost:9099`

Go has some crazy dependcy structure that mandates absolute paths and makes forking a version of a repo somewhat tricky, since you have to go and update all the dependency links.  I've tried to simplify this with python scripts, so that if we want to pull in changes from upstream, our lives shouldn't be too difficult.

Basically eth-go is a highly modular library for a full ethereum node, and go-ethereum co-ordinates the library into startup/shutdown routines convenient for the headless and gui clients. We use go-ethereum because it has some nice helper functions for startup/shutdown.

## Notes

We're using a custom blockchain with two addresses and lots of funds in each.  The keys are in `keys.txt` and both are loaded. You can get a new key with `c3d-go --newKey`.  The next time `c3d-go` starts, it will send the new address funds from a genesis addr. See `flags.go` for all the options.

To mine, `c3d-go --mine`. The difficulty is low, the logging level high.  

To play with the chat feature, `go get github.com/ebuchman/p2p_go` and run `p2p_go` in another terminal window. Once you start `c3d` and load the webapp, chat will start.


## Features

c3d-go doesn't do much yet.  It stores an infohash in a contract, waits for it to be mined, grabs the infohash from the blockchain, and throws it into the torrent client.  You can monitor the torrent client at `http://localhost:9091`. A webapp (`http://localhost:9099`) is in the works. She is a bare bones interface for doing ethereum things (txs, contract creation, storage lookup), and now also contains a barebones implementation of p2p chat, soon to be encrypted and using blockchain for authentication.

Stay tuned ...

## Contributing

1. Fork the repository.
2. Create your feature branch (`git checkout -b my-new-feature`).
3. Add Tests (and feel free to help here since I don't (yet) really know how to do that.).
4. Commit your changes (`git commit -am 'Add some feature'`).
5. Push to the branch (`git push origin my-new-feature`).
6. Create new Pull Request.

## License

Modified MIT, see LICENSE.md
