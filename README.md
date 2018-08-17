# go-sdk
Liqpay SDK for golang

# Installation and usage
```bash
go get github.com/liqpay/go-sdk
```
## Requesting
Example to retrieve all received payments
```go
import "github.com/liqpay/go-sdk"

var c = liqpay.New("your_pub_key", "your_priv_key", nil)

func main() {
    r := liqpay.Request{
    	"action": "reports",
    	"version": 3,
        "date_from": 1443161386000,
        "date_to": 1443164386000,
    }
    resp, err := c.Send("request", r)
    if err != nil {
    	fmt.Printf("error %v\n", err.Error())
    }
    fmt.Printf("response %#v\n", resp)
}
```
## 
For more info about Liqpay visit https://www.liqpay.ua
