## get terminal websocket address

1. 请求地址

```
Http Get http://terminal.wss.kfcoding.com/api/v1/pod/{namespace}/{pod-name}/shell/{container-name}
```

2. websockek连接(sockjs)

```
        sock = new SockJS('http://terminal.wss.kfcoding.com/api/sockjs?' + response.id);
        sock.onopen = function () {
            console.log('open');
            sock.send(JSON.stringify({'Op': 'bind', 'SessionID': response.id}));
            //sock.send(JSON.stringify({'Op': 'resize', 'Cols': 154, 'Rows': 39}));
        };

        sock.onmessage = function (evt) {
            console.log();
            let msg = JSON.parse(evt.data);
            switch (msg['Op']) {
                case 'stdout':
                    console.log(msg['Data']);
                    break;
                case 'toast':
                    console.log(msg['Data']);
                    break;
                default:
                    console.error('Unexpected message type:', msg);
            }
        };

        sock.onclose = function () {
            console.log('close');
        };
```
