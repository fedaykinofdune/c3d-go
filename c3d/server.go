package c3d

import (
    "github.com/project-douglas/eth-go"
    "github.com/project-douglas/eth-go/ethpub"
    "github.com/project-douglas/eth-go/ethutil"
    "github.com/project-douglas/eth-go/ethcrypto"
    "code.google.com/p/go.net/websocket"    
    "net/http"
    "html/template"
    "strings"
    "log"
    "encoding/json"
    "strconv"
    "io/ioutil"    
)

/* 
   Every connection to this server spawns a Session. Session's can be linked to particualr UIs with uiID (browser page)
   The uiID is used to determine the origin of a websocket msg, to send it to the right session
*/

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

    uiID string // link to page in browser
}

type Config struct{
    EthPort string
    EthDataDir string
    EthLogFile string
    EthConfigFile string
    EthKeyFile string
}

//paths
var templates = template.Must(template.ParseFiles(Home+"views/index.html", Home+"views/config.html", Home+"views/chat.html"))

func renderTemplate(w http.ResponseWriter, tmpl string, p interface{}){
    //we already parsed the html templates
    err := templates.ExecuteTemplate(w, tmpl+".html", p)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

// subscribe to all accounts, issue response-GetAccounts whenever there are changes
func (s *Session) accountsReactor(){
    ch := make(chan ethutil.React)
    reactor := s.ethereum.Reactor()
    for _, a := range s.Accounts{
        event := "object:"+string(ethutil.Hex2Bytes(a.Addr))
        reactor.Subscribe(event, ch) // subscribe channel to all accounts
    }
    go func(){
        for {
            _ = <- ch
            updateSession(s)
            s.handleGetAccounts(s.ethWebSocket)
        }
    }()
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

func (g *Globals) loadSession(peth *ethpub.PEthereum, ethereum *eth.Ethereum) *Session {
     keyRing := g.keyMang.KeyRing()
     session := &Session{}
     g.n_sessions += 1
     g.sessions = append(g.sessions, session)
     g.sessionMap = make(map[string]*Session)
     session.peth = peth
     (*session).AccountMap = make(map[string]int)
     for i:=0;i<keyRing.Len();i++{
        key := keyRing.GetKeyPair(i)
        addr := ethutil.Bytes2Hex(key.Address())
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
     session.uiID = ""
    return session
}

func updateConfig(c *Config){
    //TODO
}


// render html loads
func (g *Globals) handleChat(w http.ResponseWriter, r *http.Request){
        sesh := g.loadSession(g.peth, g.eth)
        updateSession(sesh)
        sesh.accountsReactor()
        renderTemplate(w, "chat", sesh)
}

func (c *Config) handleConfig(w http.ResponseWriter, r *http.Request){
        updateConfig(c)
        renderTemplate(w, "config", c)
}

func (g *Globals) handleIndex(w http.ResponseWriter, r *http.Request){
        if r.FormValue("reset_config") == "1"{
            // reset everything with new config :)
        }
        sesh := g.loadSession(g.peth, g.eth)
        updateSession(sesh)
        sesh.accountsReactor()
        renderTemplate(w, "index", sesh)
}

// serve static files
func serveFile(w http.ResponseWriter, r *http.Request){
    if !strings.Contains(r.URL.Path, "."){
        //s.handleIndex(w, r)
    }else{
        http.ServeFile(w, r, r.URL.Path[1:])
    }
}


// WebSocket connections: chatSocketHandler, ethereumSocketHandler
// Each has a little json api

/*
    Chat API spec: {"method", "uiID", "data"}
        - "start_chat" : no params
        - "connect_peers" : ["addr", "addr", ...]
        - "send_msg" : {"to", "msg"}
*/

func (g *Globals) newHello(m map[string]interface{}, ws *websocket.Conn) bool{
    uiid := m["uiID"].(string)
    typ := m["type"].(string)
    sesh := g.sessions[g.n_sessions-1]
    sesh.uiID = uiid
    g.sessionMap[uiid] = sesh
    if typ == "chat"{
        sesh.chatWebSocket = ws
        sesh.Chat.ws = ws
    } else{
        sesh.ethWebSocket = ws
    }
    r := Response{Response:"hello"}
    by, _ := json.Marshal(r)
    websocket.Message.Send(ws, string(by))
    return true
}

func (g *Globals) chatSocketHandler(ws *websocket.Conn){
    var in []byte
    /*
    if s.chatWebSocket == nil{
        s.chatWebSocket = ws
        s.Chat.ws = ws
    } else {
        s.chatWebSocket.Close()
        s.chatWebSocket = ws
        s.Chat.ws = ws
    }*/
    hello := false
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

        if !hello &&  m["method"] != "hello"{
            break
        } else if !hello && m["method"] == "hello"{
            hello = g.newHello(m, ws)
        } else{
            log.Println(g.sessionMap)
            log.Println(m)
            s := g.sessionMap[m["uiID"].(string)]
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
}


/*
    ethereum-socket API spec:
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

func (g *Globals) ethereumSocketHandler(ws *websocket.Conn){
    var in []byte
    /*
    if s.ethWebSocket == nil{
        s.ethWebSocket = ws
    } else{
        s.ethWebSocket.Close()
        s.ethWebSocket = ws
    }*/
    hello := false
    for{
            var f interface{} // for marshaling bytes from socket through json (they may have different types)
            err := websocket.Message.Receive(ws, &in)
            if err != nil{
                log.Println("error", err)
                ws.Close()
                break
            }
            err = json.Unmarshal(in, &f)
            m := f.(map[string]interface{})
            log.Println(m)
            if !hello && m["method"] != "hello"{
                break
            } else if !hello && m["method"] == "hello"{
                hello = g.newHello(m, ws)
            } else{
                s := g.sessionMap[m["uiID"].(string)]
                if m["method"] == "transact"{
                    a := m["args"].(map[string]interface{})
                    s.handleTransact(ws, a)
                } else if m["method"] == "get_accounts"{
                    s.handleGetAccounts(ws)
                } else if m["method"] == "get_storage"{
                    a := m["args"].(map[string]interface{})
                    s.handleGetStorage(ws, a)
                } else if m["method"] == "get_src"{
                    a := m["args"].(map[string]interface{})
                    filename := a["filename"].(string)
                    handleGetSource(ws, filename)
                }

            }
    }
}

type Response struct{
    Response string
    Data map[string]string
}

func handleGetSource(ws *websocket.Conn, filename string){
    file, err := ioutil.ReadFile(filename)
    r := Response{Response:"get_src", Data:make(map[string]string)}
    if err != nil{
        r.Data["error"] = err.Error()
    } else{
        r.Data["error"] = ""
        r.Data["contents"] = string(file)
        r.Data["filename"] = filename
    }
    by, _ := json.Marshal(r)
    log.Println("sending get src", string(by))
    websocket.Message.Send(ws, string(by))
}


func (s *Session) handleGetAccounts(ws *websocket.Conn){
    acc := Response{Response:"get_accounts", Data:make(map[string]string)} 
    for _, a := range s.Accounts{
        acc.Data[a.Addr] = a.Value
    }
    by, _ := json.Marshal(acc)

    if ws != nil{
        websocket.Message.Send(ws, string(by))
    }
}

// this needs proper error handling and to be able to send back on the socket the problem
func (s *Session) handleTransact(ws *websocket.Conn, tx map[string]interface{}){
        log.Println("handle")
        from := tx["from_addr"].(string)
        recipient := tx["recipient"].(string)
        amount := tx["amount"].(string)
        gas := tx["gas"].(string)
        gasP := tx["gasprice"].(string)
        data := tx["data"].(string)
        lang := tx["script_lang"].(string)
        acc_num := (*s).AccountMap[from]
        priv := (*s).Accounts[acc_num].Priv
        log.Println(recipient, amount, gas, gasP, from, acc_num)

        var p *ethpub.PReceipt
        var err error
        if recipient == ""{
            if lang == "lll"{
                data = CompileLLL(data)
                if data == ""{
                    // failed to compile
                }
                log.Println("compiled lll")
                log.Println(data)
                log.Println(len(data))
                log.Println([]byte(data))
            }
            p, err = (*s).peth.Create(ethutil.Bytes2Hex(priv), amount, gas, gasP, data)
            if err != nil{
                log.Println(err)
            }
        } else{
            p, err = (*s).peth.Transact(ethutil.Bytes2Hex(priv), recipient, amount, gas, gasP, data)
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

type Globals struct {
    peth *ethpub.PEthereum
    eth *eth.Ethereum
    keyMang *ethcrypto.KeyManager

    sessions []*Session
    n_sessions int
    sessionMap map[string]*Session
}

func StartServer(peth *ethpub.PEthereum, ethereum *eth.Ethereum, keyMang *ethcrypto.KeyManager){
    conf := loadConfig(peth)
    g := Globals{peth:peth, eth:ethereum, n_sessions:0, keyMang:keyMang}

    // pages
    http.HandleFunc("/", g.handleIndex) // main page
    http.HandleFunc("/chat", g.handleChat) // chat page
    http.HandleFunc("/config", conf.handleConfig) // config page
    http.HandleFunc("/assets/", serveFile) // static files

    // sockets
    http.Handle("/chat_sock", websocket.Handler(g.chatSocketHandler))
    http.Handle("/ethereum", websocket.Handler(g.ethereumSocketHandler))

    http.ListenAndServe(":9099", nil)
}
