package c3d

import (
    "github.com/project-douglas/eth-go/ethutil"
    "github.com/project-douglas/eth-go/ethpub"
    "io/ioutil"
    "strings"
    "log"
    "strconv"
)

func newKeyPair(){
    keyPair, err := ethutil.GenerateNewKeyPair()
    if err != nil{
        log.Println("Tragedy! Could not generating keypair! Ba-bye!")
        log.Fatal(err)
    }
    ethutil.GetKeyRing().Add(keyPair)
//    keyRing.NewKeyPair(keyPair.PrivateKey)
}

// private keys in plain-text hex format one per line
func LoadKeys(filename string){
    keyData, err := ioutil.ReadFile(filename)
    if err != nil{
        log.Println("Could not find keys file. Creating new keypair...")        
        newKeyPair()
    } else { 
        keys := strings.Split(string(keyData), "\n")
        for _, k := range keys{
            if len(k) == 64{
                keyPair, err := ethutil.NewKeyPairFromSec(ethutil.FromHex(k))
                if err == nil{
                    ethutil.GetKeyRing().Add(keyPair)
                }
            }
        }
    }
    if ethutil.GetKeyRing().Len() == 0{
        newKeyPair()
    }
    logger.Infoln("Keys loaded: ", ethutil.GetKeyRing().Len())
}

func CheckZeroBalance(peth *ethpub.PEthereum){
    keys := ethutil.GetKeyRing()
    master := ethutil.Hex(keys.Get(keys.Len()-1).PrivateKey)
    logger.Infoln("master has ", peth.GetStateObject(ethutil.Hex(keys.Get(keys.Len()-1).Address())).Value())
    for i:=0; i<keys.Len();i++{
        k := keys.Get(i).Address()
        val := peth.GetStateObject(ethutil.Hex(k)).Value()
        logger.Infoln("key ", i, " ", ethutil.Hex(k), " ", val)
        v, _ := strconv.Atoi(val)
        if v < 100 {
            _, err := peth.Transact(master, ethutil.Hex(k), "10000000000000000000", "1000", "1000", "")
            if err != nil{
                logger.Infoln("Error transfering funds to ", ethutil.Hex(k))
            }
        }
    }
}

