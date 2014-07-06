package c3d

import (
    "github.com/project-douglas/eth-go/ethutil"
    "os/exec"
)


func BigNumStrToHex(s string) string{
    bignum := ethutil.Big(s)
    bignum_bytes := ethutil.BigToBytes(bignum, 16)
    return ethutil.Hex(bignum_bytes)
}

func KillPidByName(name string){
    /* should be cross platform
       `ps aux | grep name | awk '{print $2}' | xargs kill -9`
    */
    c1 := exec.Command("ps", "aux")
    c2 := exec.Command("grep", name)
    c3 := exec.Command("awk", "{print $2}")
    c4 := exec.Command("xargs", "kill", "-9")
    c2.Stdin, _ = c1.StdoutPipe()
    c3.Stdin, _ = c2.StdoutPipe()
    c4.Stdin, _ = c3.StdoutPipe()
    
    c1.Start()
    c2.Start()
    c3.Start()
    c4.Start()
    c1.Wait()
    c2.Wait()
    c3.Wait()
    c4.Wait()
}
