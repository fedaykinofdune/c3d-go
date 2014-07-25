package c3d

import (
    "fmt"
    "log"
    "time"
    "os/exec"
    "bytes"
    "strings"
    "net/http"
    "errors"
    "path"
    "os"
    "io"
    "io/ioutil"
    "encoding/json"
)

/*
    Util level items
*/

var url string = fmt.Sprintf("http://localhost:%s/transmission/rpc/", *TransmissionPort)
var fields = `["id", "name", "totalSize", "isFinished", "percentDone"]`

// resolve the types for a map from string to interface{}
func resolveTypes(c map[string]interface{}) map[string]interface{}{
    r := make(map[string]interface{})
    for k, _ := range c{
        switch c[k].(type){
            case int:
                r[k] = c[k].(int)
            case string:
                r[k] = c[k].(string)
            case float64:
                r[k] = c[k].(float64)
            default:
                r[k] = c[k]
        }
    }
    return r
}

/*
    Daemon level functions
*/

// start the transmission daemon
func startTransmission(){
    cmd := exec.Command("transmission-daemon", "--watch-dir", *BlobDir, "--download-dir", *BlobDir)
    err := cmd.Run()
    if err != nil{
        logger.Infoln("Couldn't start transmission...")
        log.Fatal(err)
    }
    logger.Infoln("Successfully started Transmission.  Watch it at http://localhost:9091")
}

// check is transmission process is running
func IsTransmissionRunning() bool{
    cmd := exec.Command("pgrep", "-l", "transmission")
    var out bytes.Buffer
    cmd.Stdout = &out
    cmd.Run()
    return len(out.String()) > 0
}

// check if transmission process is running.  if not, start it
func CheckStartTransmission(){
    if !IsTransmissionRunning(){
        log.Println("Transmission not running.  Starting now...")
        startTransmission()
    }
    time.Sleep(time.Second)
}

// shutdown transmission process
func ShutdownTransmission(){
    _, err:= httpPost(url, make(map[string]string),` {"method":"session-close"}`)
    if err != nil{
        logger.Infoln("Could not shutdown transmission ", err)
    } else{
        logger.Infoln("Shutdown transmission")
    }
}

/*
    Torrent level functions
*/

// return list of all torrents with fields info
func GetTorrents() []map[string]interface{}{
    log.Println("call get torrents")
    args := fmt.Sprintf(`{"fields":%s}`, fields)
    r := transmissionRPC("torrent-get", args)
    a := r["torrents"].([]interface{})
    var t []map[string]interface{}
    for _, k := range a{
        c := k.(map[string]interface{})
        b := resolveTypes(c)
        // how to resolve arg types?!
        t = append(t, b)
    }
    return t
}

// return single torrent with fields info
func GetTorrent(id int) map[string]interface{}{
    log.Println("get torrent")
    args := fmt.Sprintf(`{"fields":%s, "ids":[%d]}`, fields, id)
    r := transmissionRPC("torrent-get", args)
    log.Println(r)
    return r
}

// return id of torrent
func GetTorrentID(file string) int{
    ts := GetTorrents()
    for _, t := range ts{
        if t["name"] == file{
            return t["id"]
        }
    }
    return -1
}


// create new torrent from file, save in BlobDir, return path
func CreateTorrent(filename string) string{
    _, file := path.Split(filename)
    newpath := path.Join(*BlobDir, file+".torrent")
    cmd := exec.Command("transmission-create", filename, "--outfile", newpath)
    err := cmd.Run()
    if err != nil{
        logger.Infoln("Couldn't create torrent from ", filename)
        return ""
    } else{
        logger.Infoln("Successfully created new torrent file")
        return newpath 
    }
}

// start a torrent in transmission
func StartTorrent(file string){
    file = checkFileOrHash(file)
    args := fmt.Sprintf(`{"download-dir":"%s", "filename":"%s"}`, *BlobDir, file)
    r := transmissionRPC("torrent-add", args)
    log.Println(r)
    if r != nil{
        logger.Infoln("Torrent start unsuccessful")
    } else {
        logger.Infoln("Successfully started torrent ", file, ". Monitor its progress at http://localhost:9091")
    }
}

// stop torrent.  TODO: modify to deal with filename/infohash
func StopTorrent(file string){
    file = checkFileOrHash(file)
    args := fmt.Sprintf(`{"filename":"%s"}`, file)
    r := transmissionRPC("torrent-stop", args)
    if r != nil{
        logger.Infoln("Torrent stop unsuccessful")
    } else {
        logger.Infoln("Successfully stopped torrent ", file, ".")
    }
}

// turn a file into a torrent and start seeding
func NewBlob(filepath string){
    newpath := CreateTorrent(filepath)
    if newpath != ""{
        StartTorrent(newpath)     
    }
}

// check if string is a path or just an infohash
// if infohash, convert to magnetlink
func checkFileOrHash(file string) string {
    link := ""
    if path.Ext(file) != ""{
        _, err := os.Stat(file)
        if err != nil{
            return ""
        }
        link = file
    } else {
        link = fmt.Sprintf("magnet:?xt=urn:btih:%s",file)
    }
    return link
}

/*
    RPC level funtions
*/

// for general communication with transmission rpc.
// takes a reference to a file (name or infohash), and applies a method.  no args yet
func transmissionRPC(method, args string) map[string]interface{} {
    //json := fmt.Sprintf(`{"arguments":{"filename":"%s"}, "method": "%s"}`, link, method)
    body := fmt.Sprintf(`{"arguments":%s, "method": "%s"}`, args, method)
    header := make(map[string]string)
    log.Println(body)
    m, err := httpPost(url, header, body)
    if err != nil{
        log.Println("error in http post:", err)
    }
    return m
}

// for making http requests through rpc.
// deals with 409 errors (establishing session-id)
// returns response arguments
func httpPost(url string, header map[string]string, body string) (map[string]interface{}, error) {
    b := strings.NewReader(body)
    client := &http.Client{}

    // create request
    req, err := http.NewRequest("POST", url, b)
    if err != nil{
        logger.Infoln(err)
    }
    // add header info
    for k, v := range header{
        req.Header.Add(k, v)
    }
    // submit request
    resp, err := client.Do(req)
    if err != nil{
        logger.Infoln(err)
    }

    // initialize return
    var args map[string]interface{} = nil

    // process response
    if strings.Contains(resp.Status, "409"){
        header["X-Transmission-Session-Id"] = resp.Header["X-Transmission-Session-Id"][0]
        args, err := httpPost(url, header, body)
        return args, err
    } else if !strings.Contains(resp.Status, "200"){
        logger.Infoln("Could not connect!")
        logger.Infoln(resp)
        err = errors.New(resp.Status)
    } else {
        args = unrollArgsResult(resp.Body)
    }
    return args, err
}

// read http response.body into byte array
func unrollResponse(body io.ReadCloser) []byte{
     defer body.Close()
     contents, err := ioutil.ReadAll(body)
     if err != nil {
         log.Println("Could not read from http response body", err)
         os.Exit(1)
     }
     return contents
}

// unroll http response and parse json (surface)
func unrollArgsResult(body io.ReadCloser) map[string]interface{}{
    contents := unrollResponse(body)
    var r map[string]interface{}
    err :=  json.Unmarshal(contents, &r)
    if err != nil{
        log.Fatal("Couldn't unmarshal transmission response", err)
    }
    args := r["arguments"].(map[string]interface{})
    result := r["result"].(string)
    if result != "success"{
        log.Println(result)
    }
    return args
}






/*
    Deprecated...
*/

// live torrent
func GetTorrentInfo(infohash string) []string{
    cmd := exec.Command("transmission-remote", "--torrent", infohash, "--info")
    var out bytes.Buffer
    cmd.Stdout = &out
    err := cmd.Run()
    if err != nil {
        logger.Infoln("Couldn't get info for", infohash)
        logger.Infoln(err)
    }
    outstr := strings.Split(out.String(), "\n")
    return outstr
}

func IsTorrentDone(infohash string) bool {
    outstr := GetTorrentInfo(infohash)
    donestr := ""
    for _, o := range(outstr){
        if strings.Contains(o, "Done"){
            donestr = o
            break
        }
    }
    logger.Infoln(donestr)
    if strings.Contains(donestr, "100"){
        return true
    }
    return false
}

func StartTorrentCmd(infohash string){
    logger.Infoln("Starting torrent with infohash", infohash)
    cmd := exec.Command("transmission-remote", "--add", "magnet:?xt=urn:btih:"+infohash, "--dht")
    err := cmd.Run()
    if err != nil {
        logger.Infoln("Error! Couldn't start torrent", infohash)
    } else {
        logger.Infoln("torrent download successfully started. Monitor at http://localhost:9091")
    }
}
