package c3d

import (
    "log"
    "net"
    "fmt"
    "code.google.com/p/go.net/websocket"    
)

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

// each peer has a read_ch for reading messages in from that peer and a write chan for writing out
// broadcast loops through all of them and sends the same message
type Peer struct{
    nick string
    addr string
    conn net.Conn
    write_ch chan []byte
    quit bool
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
        c.read_ch <- con.RemoteAddr().String() + ": " + string(buf[:n])
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

type Chat struct{
    peer_ch chan net.Conn
    quit_ch chan Peer
    write_ch chan string
    read_ch chan string
    Peers map[string]Peer
    ws *websocket.Conn
}

func (c *Chat) StartChat(){
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
