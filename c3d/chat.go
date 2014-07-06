package c3d

import (
    "log"
    "net"
    "fmt"
    "code.google.com/p/go.net/websocket"    
    "encoding/json"
)

/*
P2P Chat with blockchain based authentication
- how do Alice and Bob exchange ip:ports to establish a chat session? 
- Whenever Alice adds a new friend, she sends her "friend-key".  When Alice comes online, she encrypts her "ip:port" with the friend-key and sends in a tx to her "percy-proxy".  Users subscribe to all their friends "percy-proxies"
- Bob will see Alice sent a tx to her percy-proxy, and will decrypt the msg with Alice's friend key, then try to connect to her

- once we have peer connections, we can build conversations.
- communication with byte streams: "method" (1 byte), "convo_id" (31 bytes), "data" (arbitrary) 
    - methods:
        - handshake ...
        - "msg" : length of message (1 byte), message (N bytes)
        - "invite" : number of peers (1 bytes), peers (N*6 bytes)
        - "join" : empty

- "invite" is sent to request that a peer joins a conversation. That peer will first make sure they are connected to all peers in the convo, and then send each a "join" msg with the convo id. Receivers of a "join" message will add the sender to the appropriate convo. The convo history can be cached and uplodaed via bittorrent.

- we make a single tcp connection to each peer, and establish a symmetric key for that session
- each conversation has a single symmetric key for encryption
- "method" and "convo_id" are encrypted (together) with the symmetric key for the peer
- for "invite", the entire thing is encrypted with the symmetric key for the peer
- messages are encrypted with the symmetric key for the convo

much to be implemented....

*/


type Peer struct{
    nick string // nickname
    addr string // ip:port
    pubkey []byte // public key
    symkey []byte // symmetric key
    conn net.Conn // tcp sockey
    write_ch chan []byte // write stream of bytes to peer
    quit bool
}

type Conversation struct{
    Peers map[string]Peer // peers in the conversation
    write_ch chan string // input from me on this conversation (write to peers)
    read_ch chan string // input from peers in this conversation (read from peers)
    key []byte // conversation key
}

type Chat struct{
    peer_ch chan net.Conn // new peers from listenServer
    quit_ch chan Peer // quit a peer
    write_ch chan string // input from me (write to peers)
    read_ch chan string // input from peers (read from peers)
    Peers map[string]Peer // {"ip:port" : Peer}
    Conversations []Conversation // list of active conversations
    ws *websocket.Conn // websocket for writing straight to browser
    started bool
}


// listen on port. Accept connections.  Broadcast new connection on peer_channel (dealt with by PeerManager)
func (c *Chat) listenServer(){
    ln, err := net.Listen("tcp", ":"+*ChatPort)
    if err != nil{
        log.Fatal(err)
    } else {
        log.Println("chat listening on ", *ChatPort)
    }
    for {
        conn, err := ln.Accept()
        fmt.Println(conn)
        if err!= nil{
            continue
        }
        c.peer_ch <- conn
    }
}


func (c *Chat) readPeer(me *Peer){
    buf := make([]byte, 1024)
    con := (*me).conn
    // read loop - try and read over socket, print result to terminal
    for{
        n, err := con.Read(buf)
        if err != nil || n == 0{
            (*me).quit = true
            c.quit_ch <- *me
            break
        }
        fmt.Println(con.RemoteAddr().String() + ": " + string(buf[:n]))
        r := Response{Response:"msg", Data:make(map[string]string)}
        r.Data["from"] = con.RemoteAddr().String() 
        r.Data["msg"] = string(buf[:n])
        by , _ := json.Marshal(r)
        log.Println(string(by))
        c.read_ch <- string(by)
    }
}

func (c* Chat) writePeer(me *Peer){
    con := me.conn
    // write loop.  If something comes in on me.ch, write msg to peer
    for {
        buf := <- (*me).write_ch   
        n, err := con.Write(buf)
        if err!=nil || n == 0{
            (*me).quit = true
            c.quit_ch <- *me
            break
        }
    }
}

// each peer has a PeerLoop with two concurrent funcs, one for reading, one for writing.
func (c *Chat) peerLoop(me Peer){
    go c.readPeer(&me)
    go c.writePeer(&me)
}

// connect to peers from command line args.
// this should expand to read known peers from a file
func (c *Chat) connectPeers(){

   //conn, err := net.Dial("tcp", *remote_host + ":" + *remote_port)
   conn, err := net.Dial("tcp", "localhost:3333")
   if err != nil{
        fmt.Println(err)   
   }else{
       this_peer := Peer{nick:"jim", addr:conn.RemoteAddr().String(), conn:conn, write_ch:make(chan []byte), quit:false}
       c.Peers[this_peer.addr] = this_peer
       //go volleyUp(this_peer, c.read_ch)
       go c.peerLoop(this_peer)
   }
}

func (c *Chat) ConnectPeers(peers []string){
    for _, p := range peers{
        conn, err := net.Dial("tcp", p)
        if err != nil{
            log.Println("Could not connect to ", p)
            continue
        }
        this_peer := Peer{nick:"jim", addr:conn.RemoteAddr().String(), conn:conn, write_ch:make(chan []byte), quit:false}
        c.Peers[this_peer.addr] = this_peer
        go c.peerLoop(this_peer)
    }
}

// When a new peer connects, create Peer and add to peers
func addNewPeer(peers *map[string]Peer, conn net.Conn) Peer {
    this_peer := Peer{nick:"jim", addr:conn.RemoteAddr().String(), conn:conn, write_ch:make(chan []byte), quit:false}
    fmt.Println("New Peer!", conn.RemoteAddr())
    (*peers)[conn.RemoteAddr().String()] = this_peer
    return this_peer
}

// close a peer connection and remove from peers
func closePeer(peers *map[string]Peer, peer *Peer){
    addr := (*peer).addr
    (*peers)[addr].conn.Close()
    delete((*peers), addr)
    fmt.Println("closing peer", *peer)
}

func (c *Chat) WritePeer(to, msg string){
    broadcastMsg(c.Peers, msg)
}

// send a message to all peers by writing to their channels
func broadcastMsg(peers map[string]Peer, msg string){
    for _, p := range peers{
        //fmt.Println("sending  to ", p.addr, "this msg", msg)
        p.write_ch <- []byte(msg)
    }
}

// manage peers: 1) accept new peers. 2) broadcast new writes. 3) broadcast new std inputs
func (c *Chat) peerManager(){
    c.connectPeers()
    for {
        select{
        case new_peer := <- c.peer_ch:
            if new_peer != nil{
                this_peer := addNewPeer(&c.Peers, new_peer)
                go c.peerLoop(this_peer)
            }
        case to_quit := <- c.quit_ch:
           closePeer(&c.Peers, &to_quit)
        case input := <- c.write_ch:
           broadcastMsg(c.Peers, input) 
        case read := <- c.read_ch:
            // write read to websocket. let javascript parse person
            websocket.Message.Send(c.ws, string(read))
        }
        
    }
}

// for now, we can only start one chat (ie can't have multiple browser windows trying to start chat
// should be ok, and can make multiple conversations
func (c *Chat) StartChat(){
    if !c.started{
        c.started = true
        c.peer_ch = make(chan net.Conn) // for serving new peers from the tcpServer
        defer close(c.peer_ch)
        c.quit_ch = make(chan Peer) // signal to close a peer
        defer close(c.quit_ch)
        c.write_ch = make(chan string) // :/c.
        defer close(c.write_ch)
        c.read_ch = make(chan string) // input from stdin
        defer close(c.read_ch)

        c.Peers = make(map[string]Peer)

        go c.listenServer()
        c.peerManager()
    }
}
