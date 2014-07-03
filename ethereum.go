
package main

import (
    "github.com/ethereum/eth-go"
    "github.com/ethereum/eth-go/ethutil"
    "github.com/ethereum/eth-go/ethpub"
    "github.com/ethereum/eth-go/ethlog"
    "github.com/ethereum/go-ethereum/utils"
    "log"
    "bytes"
    "fmt"
)

//Logging
var logger *ethlog.Logger = ethlog.NewLogger("C3D")


func EthConfig() {
    ethutil.ReadConfig(*ethConfigFile, *ethDataDir, "", "c3d-go")
    utils.InitLogging(*ethDataDir, *ethLogFile, int(ethlog.DebugLevel), "")
}

func NewEthPEth() (*eth.Ethereum, *ethpub.PEthereum){
    // create a new ethereum node: init db, nat/upnp, ethereum struct, reactorEngine, txPool, blockChain, stateManager
    ethereum, err := eth.New(eth.CapDefault, false)
    if err != nil {
        log.Fatal("Could not start node: %s\n", err)
    }
    // initialize the public ethereum object. this is the interface QML gets, and it's mostly good enough for us to
    peth := ethpub.NewPEthereum(ethereum) 
    return ethereum, peth
}

func GetStorageAt(peth *ethpub.PEthereum, contract_addr string, storage_addr string) string{
    ret := peth.GetStorage(contract_addr, storage_addr) // returns a massive base-10 integer.
    val := BigNumStrToHex(ret)
    return val
}

func GetInfoHashStartTorrent(peth *ethpub.PEthereum, contract_addr string, storage_addr string){
    logger.Infoln("contract addr and storage ", contract_addr, storage_addr)
    infohash := GetStorageAt(peth, contract_addr, storage_addr)
    logger.Infoln("recovered infohash ", infohash)
    StartTorrent(infohash)
}

func CurrentInfo(peth *ethpub.PEthereum){
    n_peers := peth.GetPeerCount()
    peers := peth.GetPeers()
    addr := peth.GetKey().Address
    state := peth.GetStateObject(addr)
    isMin := peth.GetIsMining()
    isLis := peth.GetIsListening()
    //coinbase := peth.GetCoinBase()
    txs := peth.GetTransactionsFor(addr, true)
    //tx_count := peth.GetTxCountAt(addr)
    
    var buf bytes.Buffer
    buf.WriteString("Summary of Current Ethereum Node State\n")
    buf.WriteString(fmt.Sprintf("\tN Peers: \t %d\n", n_peers))
    for _, p := range(peers){
        buf.WriteString(fmt.Sprintf("\t\t\tPeer: %s:%s\n", p.Ip,  p.Port))
    }
    buf.WriteString("\tTop Address on KeyRing:\n")
    buf.WriteString(fmt.Sprintf("\t\tAddress:\t %s\n", addr))
    buf.WriteString(fmt.Sprintf("\t\tValue:\t %s\n", state.Value()))
    buf.WriteString(fmt.Sprintf("\t\tNonce:\t %d\n", state.Nonce()))
    buf.WriteString(fmt.Sprintf("\t\tIs Mining?\t %t\n", isMin))
    buf.WriteString(fmt.Sprintf("\t\tIs Listening?\t %t\n", isLis))
    buf.WriteString(fmt.Sprintf("\t\tTxs for\t %s\n", txs))
    //buf.WriteString("Coinbase: \t", coinbase)
    logger.Infoln(buf.String())
}

