package c3d

import (
    "log"
    "math"
    "crypto/sha1"
    "io/ioutil"
    "strings"
    "bytes"
    "os"
    "fmt"
    "code.google.com/p/bencode-go"
)

var PIECE_SIZE = int(math.Pow(2, 18))

// wrap a file in a torrent
type TorrentFile struct{
    Announce string "announce"
    Info InfoDictFile "info"
}

// wrap a dir in a torrent
type TorrentDir struct{
    Announce string "announce"
    Info InfoDictDir "info"
}

// info dict for a single file
type InfoDictFile struct{
    Name string "name" // name of file
    PieceLength int "piece length" // uniform length of pieces
    Pieces string "pieces" // sha1 hash of every piece
    Length string "length" // length of file in bytes
}

// info dict for multi files
type InfoDictDir struct{
    Name string "name" // name of dir
    PieceLength int "piece length"
    Pieces string "pieces"
    Files []FileDict "files" // one dictionary per file
}

// file dictionary (one per file)
type FileDict struct{
    Length int "length" // length of file in bytes
    Path []string "path" // list of subdirectory paths to file (last entry is filename)
}

// take a list of files and build a torrent from them
func CreateTorrentDir(files []string, name string){
    // read all files into a single byte buffer in memory
    // crunch through taking sha1s of pieces
    sha1s := "" // byte string, multiple of 20
    data_bytes := []byte{} // byte string of all torrent contents in memory

    T := TorrentDir{}
    Info := InfoDictDir{ Name: name, PieceLength: PIECE_SIZE}
    fileDicts := []FileDict{}

    // loop through files, read into byte string, form fileDict 
    for _, f := range files{
        b, err := ioutil.ReadFile(f)
        if err != nil{
            log.Println("failed to open file", f, err)
            return
        }
        f_dict := FileDict{Length:len(b), Path:strings.Split(f, "/")}
        fileDicts = append(fileDicts, f_dict)
        data_bytes = append(data_bytes, b...)
    }
    Info.Files = fileDicts

    // read through the byte buffer one PIECE_SIZE at a time
    n_pieces := (len(data_bytes) + PIECE_SIZE - 1) / PIECE_SIZE
    for i:=0; i<n_pieces; i++{
        L := len(data_bytes)
        // piece length is either PIECE_SIZE or its the final piece and less
        var piece_size int
        if L > PIECE_SIZE{
            piece_size = PIECE_SIZE
        } else{
            piece_size = L
        }
        // append hash to sha1s string
        hash := sha1.Sum(data_bytes[:piece_size])
        sha1s += string(hash[:])
        // remove piece_size bytes just read
        data_bytes = data_bytes[piece_size:]
    }
    Info.Pieces = sha1s
    T.Info = Info
    // torrent object complete. now bencode
    bencoded_info := new(bytes.Buffer)
    err := bencode.Marshal(bencoded_info, Info)
    if err !=nil{
        log.Println("failed to bencode info dict")
        return
    }
    // btih is sha1 of the bencoded infodict
    infohash := sha1.Sum(bencoded_info.Bytes())
    T.Announce = string(infohash[:]) // letting the announce string be the btih. we can do better tho

    // compose torrent file by bencoding torrent object and save to disk
    torrent := new(bytes.Buffer)
    err = bencode.Marshal(torrent, T)
    if err != nil{
        log.Println("could not bencode torrent", err)
    }
    err = ioutil.WriteFile(name, torrent.Bytes(), 0644)
    if err != nil{
        log.Println("Could not write torrent file!", err)
    }
}

