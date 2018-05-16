## kfcoding shell server

#### 1. 安装

#### 2. 接口说明

获得websocket地址分两个步骤：

1. 请求token
- url: api/v1/pod/{namespace}/{pod-name}/shell/{container-name}
- method: GET
- response格式: {"id" : "token"}
如：{
      "id": "42efbc2fee67b82e2278850593965255"
    }

2. websockek连接(sockjs)

```
        sock = new SockJS('api/sockjs?' + response.id);
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
