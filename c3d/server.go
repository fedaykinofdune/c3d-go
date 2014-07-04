package c3d

import (
    "github.com/ethereum/eth-go"
    "github.com/ethereum/eth-go/ethpub"
    "github.com/ethereum/eth-go/ethutil"
    "code.google.com/p/go.net/websocket"    
    "net/http"
    "html/template"
    "strings"
    "log"
    "encoding/json"
    "os"
)

type account struct{
    Addr string
    Priv []byte
    Value string
    Nonce int   
    Storage map[string]string // maps hex addrs to hex values
    Code []byte
}

type torrent struct{
    InfoHash string
    Done bool
    Contract string
}

type Session struct{
    Accounts []account
    AccountMap map[string]int //map from addr to account number
    Contracts []account
    Torrents []torrent
    peth *ethpub.PEthereum
    ethereum *eth.Ethereum
    websocket *websocket.Conn 
}

type Config struct{
    EthPort string
    EthDataDir string
    EthLogFile string
    EthConfigFile string
    EthKeyFile string
}

var templates = template.Must(template.ParseFiles("views/index.html", "views/config.html"))

func renderTemplate(w http.ResponseWriter, tmpl string, p interface{}){
    //we already parsed the html templates
    err := templates.ExecuteTemplate(w, tmpl+".html", p)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func updateSession(s *Session){
    for i:=0; i<len(s.Accounts);i++{
        addr := (*s).Accounts[i].Addr
        state := (*s).peth.GetStateObject(addr)
        val := state.Value()
        nonce := state.Nonce()
        (*s).Accounts[i].Value = val
        (*s).Accounts[i].Nonce = nonce
    }
}

func loadConfig(peth *ethpub.PEthereum) *Config{
    conf := &Config{}
    conf.EthPort = *EthPort
    conf.EthDataDir = *EthDataDir
    conf.EthLogFile = *EthLogFile
    conf.EthConfigFile = *EthConfigFile
    conf.EthKeyFile = *KeyFile

    return conf
}

func loadSession(peth *ethpub.PEthereum, ethereum *eth.Ethereum) *Session {
     keyRing := ethutil.GetKeyRing()
     session := &Session{}
     session.peth = peth
     (*session).AccountMap = make(map[string]int)
     for i:=0;i<keyRing.Len();i++{
        key := keyRing.Get(i)
        addr := ethutil.Hex(key.Address())
        priv := key.PrivateKey
        state := peth.GetStateObject(addr)
        val := state.Value()
        nonce := state.Nonce()
        ac := account{Addr: addr, Value:val, Nonce:nonce, Priv:priv}
        (*session).Accounts = append((*session).Accounts, ac)
        (*session).AccountMap[addr] = i
     }
     session.websocket = nil
     session.ethereum = ethereum
    return session
}

func (s *Session) handleTransact2(w http.ResponseWriter, r *http.Request){
        to := r.FormValue("recipient")
        val := r.FormValue("amount")
        g := r.FormValue("gas")
        gp := r.FormValue("gasprice")
        from := r.FormValue("from_addr")
        acc_num := (*s).AccountMap[from]
        priv := (*s).Accounts[acc_num].Priv
        log.Println(to, val, g, gp, from, acc_num)
        p, err := (*s).peth.Transact(ethutil.Hex(priv), to, val, g, gp, "")
        if err != nil{
            log.Println(err)
        }
        log.Println(p)
       // renderTemplate(w, "index", s)
}

func updateConfig(c *Config){
    //TODO
}

func (c *Config) handleConfig(w http.ResponseWriter, r *http.Request){
        updateConfig(c)
        renderTemplate(w, "config", c)
}

func (s *Session) handleIndex(w http.ResponseWriter, r *http.Request){
        if r.FormValue("reset_config") == "1"{
            // reset everything with new config :)
        }
        updateSession(s)
        renderTemplate(w, "index", s)
}

func (s *Session) serveFile(w http.ResponseWriter, r *http.Request){
    if !strings.Contains(r.URL.Path, "."){
        s.handleIndex(w, r)
    }else{
        //path := fmt.Sprintf("../", r.URL.Path[1:])
        http.ServeFile(w, r, r.URL.Path[1:])
    }
}

func (s *Session) webSocketHandler(ws *websocket.Conn){
    var in []byte
    if s.websocket == nil{
        s.websocket = ws
    }
    for{
            err := websocket.Message.Receive(ws, &in)
            if err != nil{
                log.Println(err)
            }
            log.Println(string(in))
            var f interface{}
            err = json.Unmarshal(in, &f)
            log.Println(f)
            m := f.(map[string]interface{})
            if m["method"] == "transact"{
                a := m["args"].(map[string]interface{})
                s.handleTransact(ws, a)
            } else if m["method"] == "get_accounts"{
                s.handleGetAccounts(ws)
            } else if m["method"] == "get_storage"{
                a := m["args"].(map[string]string)
                s.handleGetStorage(ws, a)
            }
    }
}

/*
    web-socket API spec:
        - Client Request : {"method" : ... , "args" : {   }}
            - Methods
                - transact : {"to", "value", "from", "gas", "gas_price"} 
                - get_accounts : {}
                - get_storage : {"addr", "storage"}
                - subscribe_accounts {[ac1, ac2, ...]}
                - subscribe_stores : {[{"addr", "storage"}, {"addr", "storage"}, ...] }
        - Server Response : {"response" : ... , "data" : {   }}
            - Responses
                - transact : {"success", "txid"}
                - get_accounts : {"addr":"value", "addr":"value", ... ]}
                - get_storage : {"value"}
            - Notifies
                - subscribe_accounts {}
                - subscribe stores {}
*/

func (s *Session) accountsReactor(){
    ch := make(chan ethutil.React)
    reactor := s.ethereum.Reactor()
    for _, a := range s.Accounts{
        reactor.Subscribe("object:"+string(ethutil.FromHex(a.Addr)), ch)
    }
    go func(){
        for {
            _ = <- ch
            log.Println("received signal!")
            updateSession(s)
            s.handleGetAccounts(s.websocket)
        }
    }()
}

func (s *Session) handleGetAccounts(ws *websocket.Conn){
    acc := make(map[string]interface{}) //addr:value
    acc["response"] = "get_accounts"
    acc["data"] = make(map[string]string)
    for _, a := range s.Accounts{
        (acc["data"]).(map[string]string)[a.Addr] = a.Value
    }
    by, _ := json.Marshal(acc)
    websocket.Message.Send(ws, string(by))
}

func (s *Session) handleTransact(ws *websocket.Conn, tx map[string]interface{}){
        from := tx["from_addr"].(string)
        recipient := tx["recipient"].(string)
        amount := tx["amount"].(string)
        gas := tx["gas"].(string)
        gasP := tx["gasprice"].(string)
        acc_num := (*s).AccountMap[from]
        priv := (*s).Accounts[acc_num].Priv
        log.Println(recipient, amount, gas, gasP, from, acc_num)
        p, err := (*s).peth.Transact(ethutil.Hex(priv), recipient, amount, gas, gasP, "")
        if err != nil{
            log.Println(err)
        }
        // how do I know if a tx fails?
        resp := make(map[string]interface{})
        resp["response"] = "transact"
        resp["data"] = make(map[string]string)
        resp["data"].(map[string]string)["success"] = "true"
        resp["data"].(map[string]string)["id"] = p.Hash
        by, _ := json.Marshal(resp)
        websocket.Message.Send(ws, string(by))
}

func (s *Session) handleGetStorage(ws *websocket.Conn, args map[string]string){
    addr := args["addr"]
    storage := args["storage"]
    val := GetStorageAt(s.peth, addr, storage)
    websocket.Message.Send(ws, val)
}

func StartServer(peth *ethpub.PEthereum, ethereum *eth.Ethereum){
    sesh := loadSession(peth, ethereum)
    conf := loadConfig(peth)
    sesh.accountsReactor()
    os.Exit(0)
    http.HandleFunc("/assets/", sesh.serveFile)
    http.HandleFunc("/", sesh.handleIndex)
    //http.HandleFunc("/transact", sesh.handleTransact)
    http.HandleFunc("/config", conf.handleConfig)
    http.Handle("/socket", websocket.Handler(sesh.webSocketHandler))
    http.ListenAndServe(":9099", nil)
}
