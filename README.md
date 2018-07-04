# go-sdk
Liqpay SDK for golang

1. Download release you need
2. Install golang https://golang.org/doc/install
3. Create catalog ``/src/liqpay``
4. Place ``liqpay.go`` file into ``/src/liqpay/`` folder
5. Make ``any_name.go`` file like this

``any_name.go``

```
package main

func main() {
    Init("i42883299811", "T1cBpWrXqjPFiItoQiMyXTwK88mv9baUEqviL47g")
    Api("request", map[string]interface{}{
        "action": "agent_info_merchant",
        "version": 3,
        "public_key": PublicKey,
        //place here any params you need
    })   
}
```
6. Go to ``src/liqpay/`` folder and run command 
`
go run *.go
`
7. See result in console
