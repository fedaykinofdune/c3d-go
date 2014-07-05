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
    "strconv"
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

    Chat *Chat

    peth *ethpub.PEthereum
    ethereum *eth.Ethereum

    ethWebSocket *websocket.Conn 
    chatWebSocket *websocket.Conn
}

type Config struct{
    EthPort string
    EthDataDir string
    EthLogFile string
    EthConfigFile string
    EthKeyFile string
}

var templates = template.Must(template.ParseFiles("views/index.html", "views/config.html", "views/chat.html"))

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
     session.ethWebSocket = nil
     session.chatWebSocket = nil
     session.ethereum = ethereum
     session.Chat = &Chat{}
    return session
}

func updateConfig(c *Config){
    //TODO
}

func (s *Session) handleChat(w http.ResponseWriter, r *http.Request){
        renderTemplate(w, "chat", s)
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

/*
    Chat API spec:
        - "start_chat" : no params
        - "connect_peers" : ["addr", "addr", ...]
        - "send_msg" : {"to", "msg"}
*/

func (s *Session) chatSocketHandler(ws *websocket.Conn){
    var in []byte
    if s.chatWebSocket == nil{
        s.chatWebSocket = ws
        s.Chat.ws = ws
    } else {
        s.chatWebSocket.Close()
        s.chatWebSocket = ws
        s.Chat.ws = ws
    }
    for{
        var f interface{}
        err := websocket.Message.Receive(ws, &in)
        if err != nil{
            log.Println("error:", err)
            ws.Close()
            break
        }
        err = json.Unmarshal(in, &f)
        m := f.(map[string]interface{})
        if m["method"] == "start_chat"{
            go s.Chat.StartChat()
        } else if m["method"] == "connect_new_peer"{
           peer := m["data"].(string)
           s.Chat.ConnectPeers([]string{peer}) 
        } else if m["method"] == "send_msg"{
            data := m["data"].(map[string]interface{})
            to := data["to"].(string)
            msg := data["msg"].(string)
            s.Chat.WritePeer(to, msg)
        } 
    }
}

func (s *Session) ethereumSocketHandler(ws *websocket.Conn){
    var in []byte
    if s.ethWebSocket == nil{
        s.ethWebSocket = ws
    } else{
        s.ethWebSocket.Close()
        s.ethWebSocket = ws
    }
    for{
            var f interface{} // for marshaling bytes from socket through json
            err := websocket.Message.Receive(ws, &in)
            if err != nil{
                log.Println("error", err)
                ws.Close()
                break
            }
            err = json.Unmarshal(in, &f)
            m := f.(map[string]interface{})
            if m["method"] == "transact"{
                a := m["args"].(map[string]interface{})
                s.handleTransact(ws, a)
            } else if m["method"] == "get_accounts"{
                s.handleGetAccounts(ws)
            } else if m["method"] == "get_storage"{
                a := m["args"].(map[string]interface{})
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
                - transact : {"success", "txid", "contract", "addr"}
                - get_accounts : {"addr":"value", "addr":"value", ... ]}
                - get_storage : {"value"}
            - Notifies
                - subscribe_accounts {}
                - subscribe stores {}
*/

type Response struct{
    Response string
    Data map[string]string
}

func (s *Session) accountsReactor(){
    ch := make(chan ethutil.React)
    reactor := s.ethereum.Reactor()
    for _, a := range s.Accounts{
        reactor.Subscribe("object:"+string(ethutil.FromHex(a.Addr)), ch)
    }
    go func(){
        for {
            _ = <- ch
            updateSession(s)
            s.handleGetAccounts(s.ethWebSocket)
        }
    }()
}


func (s *Session) handleGetAccounts(ws *websocket.Conn){
    acc := Response{Response:"get_accounts", Data:make(map[string]string)} //make(map[string]interface{}) //addr:value
    for _, a := range s.Accounts{
        acc.Data[a.Addr] = a.Value
    }
    by, _ := json.Marshal(acc)

    if ws != nil{
        websocket.Message.Send(ws, string(by))
    }
}

func (s *Session) handleTransact(ws *websocket.Conn, tx map[string]interface{}){
        from := tx["from_addr"].(string)
        recipient := tx["recipient"].(string)
        amount := tx["amount"].(string)
        gas := tx["gas"].(string)
        gasP := tx["gasprice"].(string)
        data := tx["data"].(string)
        acc_num := (*s).AccountMap[from]
        priv := (*s).Accounts[acc_num].Priv
        log.Println(recipient, amount, gas, gasP, from, acc_num)

        var p *ethpub.PReceipt
        var err error
        if recipient == ""{
            p, err = (*s).peth.Create(ethutil.Hex(priv), amount, gas, gasP, data)
            if err != nil{
                log.Println(err)
            }
        } else{
            p, err = (*s).peth.Transact(ethutil.Hex(priv), recipient, amount, gas, gasP, data)
            if err != nil{
                log.Println(err)
            }
        }
        // how do I know if a tx fails?
        resp := Response{Response:"transact", Data:make(map[string]string)}
        resp.Data["success"] = "true"
        resp.Data["id"] = p.Hash
        resp.Data["contract"] = strconv.FormatBool(p.CreatedContract)
        resp.Data["addr"] = p.Address
        by, _ := json.Marshal(resp)
        websocket.Message.Send(ws, string(by))
}

func (s *Session) handleGetStorage(ws *websocket.Conn, args map[string]interface{}){
    resp := Response{Response:"get_storage", Data:make(map[string]string)}
    addr := args["contract_addr"].(string)
    storage := args["storage_addr"].(string)
    val := GetStorageAt(s.peth, addr, storage)
    log.Println(addr, storage, val)
    resp.Data["addr"] = addr
    resp.Data["storage"] = storage
    resp.Data["value"] = val
    by, _ := json.Marshal(resp)
    websocket.Message.Send(ws, string(by))
}

func StartServer(peth *ethpub.PEthereum, ethereum *eth.Ethereum){
    sesh := loadSession(peth, ethereum)
    conf := loadConfig(peth)
    sesh.accountsReactor()
    http.HandleFunc("/assets/", sesh.serveFile)
    http.HandleFunc("/", sesh.handleIndex)
    //http.HandleFunc("/transact", sesh.handleTransact)
    http.HandleFunc("/config", conf.handleConfig)
    //http.HandleFunc("/chat", sesh.handleChat)
    http.Handle("/chat_sock", websocket.Handler(sesh.chatSocketHandler))
    http.Handle("/ethereum", websocket.Handler(sesh.ethereumSocketHandler))
    http.ListenAndServe(":9099", nil)
}
