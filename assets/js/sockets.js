
//websockets
window.onload = function(){
    window.state = {};
}
    
window.onload = function(){
    window.state = {}
    window.socket = new WebSocket('ws://localhost:9099/socket');

    socket.onopen = function(event) {
        var wsock = document.getElementById("websock") 
        wsock.innerHTML = "connected to "+event.currentTarget.URL;
        socket.send(JSON.stringify({"method":"get_accounts"}));
    }

    socket.onerror = function(error) {
        console.log("websocket error " + error);
    }

    socket.onmessage = function(msg){
        msg = JSON.parse(msg.data);
        data = msg["data"]
            
        var response = msg["response"];
        if (response == "transact")
            respond_transact(data);
        else if (response == "get_accounts")
            respond_get_accounts(data);
        else if (response == "get_storage")
            respond_get_storage(data);
        else if (response == "subscribe_accounts")
            respond_subscribe_accounts(data);
        else if (response == "subscribe_storage")
            respond_subscribe_storage(data);

    }
}

// make this more informative...
function respond_transact(data){
    success = data["success"];
    id = data["id"];
    console.log("succcess", success, " id", id);
}

function respond_get_accounts(data){
    for (var key in data){
        window.state[key] = data[key]
    }

    accounts = document.getElementById("accounts_dropdown");
    for (i=0; i<accounts.options.length; i++){
        accounts.options[i].value = state[accounts.options[i].innerHTML]
    }
    update_account()
}

function update_account(){
    dd = document.getElementById("accounts_dropdown");
    addr = dd.options[dd.selectedIndex].text;
    value = state[addr];
    document.getElementById("addr").innerHTML = "Current Address: " + addr;
    document.getElementById("value").innerHTML = "Value: " + value;
    document.getElementById("from_addr").value = addr;
}



function respond_get_storage(data){

}

function respond_subscribe_accounts(data){

}

function respond_subscribe_storage(data){

}

function send_tx(){
    var a = document.forms['transact_form'].getElementsByTagName('input');
    var j = {};
    j["method"] = "transact";
    j["args"] = {};
    for (i=0;i<a.length;i++){
        j["args"][a[i].name] = a[i].value;
    }
    socket.send(JSON.stringify(j));
    return false;
}
 

//socket.close();

