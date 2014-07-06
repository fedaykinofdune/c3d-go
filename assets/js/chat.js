
function setup_chat(){
    console.log("setup chat");
    window.chat_socket = new WebSocket('ws://localhost:9099/chat_sock');
    console.log(chat_socket);

    if (!("uiID" in state)){
        state["uiID"] = rando();
    }

    chat_socket.onopen = function(event) {
        console.log("opened chat socket");
        console.log(state["uiID"])
        chat_socket.send(JSON.stringify({"method":"hello", "uiID":state["uiID"], "type":"chat"}));
    }

    chat_socket.onerror = function(error) {
        console.log("websocket error " + error);
    }

    chat_socket.onmessage = function(msg){
       msg = JSON.parse(msg.data);
       data = msg["Data"];
       var response = msg["Response"];
       if (response == "hello")
           respond_chat_hello();
       else if (response == "msg")
           respond_chat_msg(data)
               
    }
}

function respond_chat_hello(){
    chat_socket.send(JSON.stringify({"method":"start_chat", "uiID":state["uiID"]}));
}

function respond_chat_msg(data){
    console.log(data);
    var from = data["from"];
    var msg = data["msg"];
        
    elem = document.getElementById("chat_div")
    elem.innerHTML += "<p>"+from+" : "+msg;
    elem.scrollTop = elem.scrollHeight;
}

function send_chat_msg(){
    var t = document.getElementById("chat_send_txt");
    var txt = t.value;
    t.value = ""
    j = {
        "method": "send_msg",
        "uiID" : state["uiID"],
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
