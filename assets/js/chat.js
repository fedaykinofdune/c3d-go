
function setup_chat(){
    console.log("setup chat");
    window.chat_socket = new WebSocket('ws://localhost:9099/chat_sock');
    console.log(chat_socket);

    chat_socket.onopen = function(event) {
        console.log("opened chat socket");
        chat_socket.send(JSON.stringify({"method":"start_chat"}));
    }

    chat_socket.onerror = function(error) {
        console.log("websocket error " + error);
    }

    chat_socket.onmessage = function(msg){
       // msg = JSON.parse(msg.data);
       // data = msg["Data"];
        console.log(msg.data);
        elem = document.getElementById("chat_div")
        elem.innerHTML += "<p>"+msg.data 
        elem.scrollTop = elem.scrollHeight;
    }
}

function send_chat_msg(){
    var t = document.getElementById("chat_send_txt");
    var txt = t.value;
    t.value = ""
    j = {
        "method": "send_msg",
        "data":{
            "to":"",
            "msg":txt
        }
    }
    chat_socket.send(JSON.stringify(j));

    elem = document.getElementById("chat_div")
    elem.innerHTML += "<p>"+"me: "+txt 
    elem.scrollTop = elem.scrollHeight;
    return false;
}
