package c3d

import (
    "github.com/project-douglas/eth-go/ethutil"
    "os"
    "os/user"
    "path"
    "flag"
    "bytes"
    "io/ioutil"
    "log"
    "fmt"
    "github.com/rakyll/globalconf"
)

func homeDir() string{
    usr, _ := user.Current()
    return usr.HomeDir
}

// Flags and Config
// Both can be specified at command line, but config can be read from $C3DDir/c3d-go.config
// cli > config-file > defaults

var GlobalConfig = make(map[string]string)
var ConfigOptions = []string{"key_file", "eth_data_dir", "eth_config_file", "eth_log_file", "eth_port", "transmission_port", 
    "c3d_dir", "c3d_config", "chat_port"}

func populateConfig(){
    GlobalConfig["key_file"] = *KeyFile
    GlobalConfig["eth_data_dir"] = *EthDataDir
    GlobalConfig["eth_config_file"] = *EthConfigFile
    GlobalConfig["eth_log_file"] = *EthLogFile
    GlobalConfig["eth_port"] = *EthPort
    GlobalConfig["transmission_port"] = *TransmissionPort
    GlobalConfig["c3d_dir"] = *C3DDir
    GlobalConfig["c3d_config"] = *C3DConfig
    GlobalConfig["chat_port"] = *ChatPort
}

var (
    kill = flag.String("kill", "", "kill a process and die")
    downloadTorrent = flag.String("downloadTorrent", "", "download torrent from infohash and die")
    isDone = flag.String("isDone", "", "check if torrent is done")
    lookupDownloadTorrent = flag.String("lookupDownloadTorrent", "", "lookup this contract address for an infohash, using storageAt flag for sotrage address")
    storageAt = flag.String("storageAt", "", "storage address in contract")
    newKey = flag.Bool("newKey", false, "create a new key and send it funds from a genesis addr")
    KeyFile = flag.String("key_file", "keys.txt", "file in which private keys are stored")
    EthDataDir = flag.String("eth_data_dir", path.Join(homeDir(), ".pd-eth"), "directory for ethereum data")
    EthConfigFile = flag.String("eth_config_file", path.Join(homeDir(), ".pd-eth/config"), "ethereum configuration file")
    EthLogFile = flag.String("eth_log_file", "", "ethereum logging file. Defaults to stdout")
    EthPort = flag.String("eth_port", "30303", "ethereum listen port")
    TransmissionPort = flag.String("transmission_port", "9091", "transmission rpc port")
    C3DDir = flag.String("c3d_dir", path.Join(homeDir(), ".c3d-go"), "directory for c3d data")
    C3DConfig = flag.String("c3d_config", path.Join(*C3DDir, "c3d-go.config"), "directory for c3d data")
    ChatPort = flag.String("chat_port", "9100", "p2p websocket chat port")
    Mine = flag.Bool("mine", false, "start mining ethereum blocks")
    Home = os.Getenv("GOPATH") + "/src/github.com/project-douglas/c3d-go/"
)

func readConfigFile(){
    _, err := os.Stat(*C3DConfig)
    if err != nil && os.IsNotExist(err){
        log.Println("No config file. Creating now")
        os.MkdirAll(*C3DDir, 0777)
        f, err := os.Create(*C3DConfig)
        if err != nil{
            log.Println("Could not create config file:", err)
        }else{
            populateConfig()
            for _, k := range ConfigOptions{
                f.WriteString(k+" = "+GlobalConfig[k]+"\n")
            }
        }
    } else{
        conf, err := globalconf.NewWithOptions(&globalconf.Options{
            Filename: *C3DConfig,
        })
        if err != nil{
            log.Println("Could not read from config file", err)
        } else{
            conf.ParseAll()
            populateConfig()
        }
    }
}


func Init(){
    readConfigFile()

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

    *KeyFile = Home + *KeyFile
    
}
