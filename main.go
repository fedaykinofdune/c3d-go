package main

import (
    "github.com/ethereum/eth-go"
    "github.com/ethereum/eth-go/ethutil"
    "github.com/ethereum/eth-go/ethpub"
    "github.com/ethereum/go-ethereum/utils"
    "github.com/project-douglas/c3d-go/c3d"
    "os"
    "log"
    "time"
)

// monitor for state change at addr using reactor. get info hash from contract, load into transmission
func callback(peth *ethpub.PEthereum, addr string, ethereum *eth.Ethereum){
    addr = string(ethutil.FromHex(addr))
    ch := make(chan ethutil.React, 1)
    reactor := ethereum.Reactor()
    reactor.Subscribe("object:"+addr, ch) // when the state at addr changes, this channel will receive
    for {
        _ = <- ch
        hexAddr := ethutil.Hex([]byte(addr))
        //c3d.logger.Infoln("hex addr ", hexAddr)
        c3d.GetInfoHashStartTorrent(peth, hexAddr, "0")
    }
}

/*
    Demonstration of simplest functionality.  
    Start everything up, stick an infohash in the blockchain, retreive it, plop into Transmission, download files over BT

*/
func main() {
    // parse flags.
    c3d.Init()

    // check if transmission is running. if not, start 'er up
    c3d.CheckStartTransmission()

    // basic ethereum config.  let's put this in a big file
    c3d.EthConfig()

    ethereum, peth := c3d.NewEthPEth()
    ethereum.Port = *c3d.EthPort
    ethereum.MaxPeers = 10

    //start the node
    ethereum.Start(false)

    // deal with keys :) the two genesis block keys are in keys.txt.  loadKeys will get them both for you.
    // if there are more keys, having 0 balance, funds will be transfered to them
    c3d.LoadKeys(*c3d.KeyFile)

    go c3d.StartServer(peth, ethereum)

    // start mining
    if *c3d.Mine{
        utils.StartMining(ethereum)
    }

    // checks if any addrs have 0 balance, tops them up
    c3d.CheckZeroBalance(peth)

   
    keyRing := ethutil.GetKeyRing()
    priv := ethutil.Hex(keyRing.Get(0).PrivateKey)
    //addrHex := ethutil.Hex(keyRing.Get(0).Address())

    //time.Sleep(time.Second*10)    

    //store an infohash at storage[0]
    infohash := "0x1183596810fbca83fce8e12d98234aaaf38eb7cd"
    p, err := peth.Create(priv, "271", "2000", "1000000", "this.store[0] = " + infohash)
    if err != nil{
        log.Fatal(err)
    }
    log.Println("created contract with address ", p.Address, " to store the infohash ", infohash)
    time.Sleep(time.Second)
    c3d.CurrentInfo(peth)

    /* The storage is not available until we've mined. We'll ultimately need access to the txPool
        for now, we use a callback that triggers when our contracts state changes
    */
    go callback(peth, p.Address, ethereum)
    ethereum.WaitForShutdown()

    os.Exit(0)

}
