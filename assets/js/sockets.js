//websockets

function rando(){
    var r = Math.random();
    var c =  r.toString();
    var d = c;
    console.log(c);
        return c;
       
}

window.onload = function(){
    window.state = {} // {"addr":"value"}

    window.eth_socket = new WebSocket('ws://localhost:9099/ethereum');
    console.log("got eth socket");
    console.log(eth_socket)


    eth_socket.onopen = function(event) {
        state["uiID"] = rando();
        eth_socket.send(JSON.stringify({"method":"hello", "uiID":state["uiID"], "type":"eth"}))
    }

    eth_socket.onerror = function(error) {
        console.log("websocket error " + error);
    }

    eth_socket.onmessage = function(msg){
        msg = JSON.parse(msg.data);
        data = msg["Data"]
        var response = msg["Response"];
        if (response == "hello")
            respond_eth_hello()
        else if (response == "transact")
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

function respond_eth_hello(){
    eth_socket.send(JSON.stringify({"method":"get_accounts", "uiID":state["uiID"]}));
}

// make this more informative...
function respond_transact(data){
    success = data["success"];
    id = data["id"];
    contract = data["contract"];
    addr = data["addr"];
    console.log("succcess", success, " id", id);
    if (contract == "true"){
        var l = document.createElement("LI");
        l.innerHTML = addr;
        document.getElementById("contract_list").appendChild(l);
    }
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
    return addr
}

function change_account(){
    var addr = update_account();
    // set sender (from_addr)  
    document.forms["transact_form"].elements['from_addr'].value = addr;
    document.forms["create_contract_form"].elements['from_addr'].value = addr;
}

function respond_get_storage(data){
    console.log(data)
    console.log(data["value"])
}

function respond_subscribe_accounts(data){

}

function respond_subscribe_storage(data){

}

// roll a form into {method, args{}} and send on socket
function make_request(form_name){
    var a = document.forms[form_name].getElementsByTagName('input');
    var j = {"uiID":state["uiID"]};
    if (form_name == "storage_lookup")
        j["method"] = "get_storage";
    else
        j["method"] = "transact";

    j["args"] = {};
    for (i=0;i<a.length;i++){
        j["args"][a[i].name] = a[i].value;
    }
    console.log(j);
    console.log(j["args"]);

    eth_socket.send(JSON.stringify(j));
    return false;
}


//socket.close();

