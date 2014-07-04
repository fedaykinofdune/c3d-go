[![Stories in Ready](https://badge.waffle.io/project-douglas/c3d-go.png?label=ready&title=Ready)](https://waffle.io/project-douglas/c3d-go)

## Introduction

C3D => Contract Controlled Content Distribution

More coming.

## How to play

To setup, you'll have to do a little mix of `go` and `git`.  Install go, setup your `$GOPATH`, and `go get github.com/ethereum/go-ethereum`. I also recommend adding `$GOPATH` to your `$PATH`, since then you can call up the executables right away from anywhere.

Now, that should put two repos in `$GOPATH/src/github.com/ethereum`.  Go into both, and use git to add the project-douglas versions as remotes:

eg.

```
cd $GOPATH/src/github.com/ethereum/eth-go
git remote add pd git@github.com:project-douglas/eth-go
git checkout -b pd
git pull pd pd
```

That adds the project-douglas repo as a remote, checkouts out a new pd branch, and pulls the pd repo down to that branch.  If you want to switch back to the real ethereum, just do `git checkout develop`

Of course, do the exact same thing for `$GOPATH/src/github.com/ethereum/go-ethereum`.

Basically eth-go is a highly modular library for a full ethereum node, and go-ethereum co-ordinates the library into startup/shutdown routines convenient for the headless and gui clients. We use go-ethereum because it has some nice helper functions for startup/shutdown.

Now, grab c3d-go: `go get github.com/project-douglas/c3d-go`. That will install it.  If you make changes and want to re-install, just hit `go install` in the c3d-go repo. Run it with `c3d-go`, or `$GOPATH/bin/c3d-go` if you must.  The webapp is at `http://localhost:9099`

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
