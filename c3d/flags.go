package c3d

import (
    "github.com/ethereum/eth-go/ethutil"
    "os"
        "os/user"
    "path"
    "flag"
    "bytes"
    "io/ioutil"
    "log"
    "fmt"
)

func homeDir() string{
    usr, _ := user.Current()
    return usr.HomeDir
}

// Flags
var (
    kill = flag.String("kill", "", "kill a process and die")
    downloadTorrent = flag.String("downloadTorrent", "", "download torrent from infohash and die")
    isDone = flag.String("isDone", "", "check if torrent is done")
    lookupDownloadTorrent = flag.String("lookupDownloadTorrent", "", "lookup this contract address for an infohash, using storageAt flag for sotrage address")
    storageAt = flag.String("storageAt", "", "storage address in contract")
    newKey = flag.Bool("newKey", false, "create a new key and send it funds from a genesis addr")
    KeyFile = flag.String("keyFile", "keys.txt", "file in which private keys are stored")
    EthDataDir = flag.String("ethDataDir", path.Join(homeDir(), ".pd-eth"), "directory for ethereum data")
    EthConfigFile = flag.String("ethConfigFile", path.Join(homeDir(), ".pd-eth/config"), "ethereum configuration file")
    EthLogFile = flag.String("ethLogFile", "", "ethereum logging file. Defaults to stdout")
    EthPort = flag.String("ethPort", "30303", "ethereum listen port")
    TransmissionPort = flag.String("transmissionPort", "9091", "transmission rpc port")
    ChatPort = flag.String("chatPort", "9100", "p2p websocket chat port")
    Mine = flag.Bool("mine", false, "start mining ethereum blocks")
)

func Init(){
    flag.Parse()
    if *kill != ""{
        KillPidByName(*kill)
        os.Exit(0)
    }
    if *downloadTorrent != ""{
        DownloadTorrent(*downloadTorrent)
        os.Exit(0)
    }
    if *isDone != ""{
        done := IsTorrentDone(*isDone)
        logger.Infoln("\tIs done:", done)
        os.Exit(0)
    }
    if *lookupDownloadTorrent != ""{
       if *storageAt == ""{
            *storageAt = "0"
       }
       EthConfig()
       _ , peth := NewEthPEth()
       GetInfoHashStartTorrent(peth, *lookupDownloadTorrent, *storageAt)
       os.Exit(0)
    }
    if *newKey{
        args := flag.Args()
        n := flag.NArg()
        filename := *KeyFile
        if n > 0{
            filename = args[0]
        }
        var buf bytes.Buffer
        keyData, err := ioutil.ReadFile(filename)
        kP, err:= ethutil.GenerateNewKeyPair()
        if err != nil{
            log.Fatal("could not generate key")
        }
        priv := kP.PrivateKey
        buf.WriteString(ethutil.Hex(priv))
        buf.WriteString("\n")
        buf.Write(keyData)
        fmt.Println(buf.String())
        err = ioutil.WriteFile(filename, buf.Bytes(), 0777)
        if err != nil{
            log.Fatal("error writing to key file")
        }
        log.Println("New key generated and added to ", filename, ". Funds will be deposited on next start up")
        os.Exit(0)
    }

}
