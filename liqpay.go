package main

import (
    "encoding/json"
    "fmt"
    "bytes"
    "io/ioutil"
    "net/http"
    "net/url"
    "crypto/sha1"
)
import b64 "encoding/base64"


var PublicKey  = ""
var PrivateKey = ""
var LiqpayURL  = "https://www.liqpay.ua/api/"


func Init(PubKey string, PrivKey string) {
    PublicKey  = PubKey
    PrivateKey = PrivKey
}


func Api(apiUrl string, Data map[string]interface{}) {
    
    DataBytes, _ := json.Marshal(Data)
    
    DataString := string(DataBytes)
    fmt.Println(DataString)

    DataBase64 := b64.StdEncoding.EncodeToString([]byte(DataString))
    fmt.Println(DataBase64)

    hasher := sha1.New()
    hasher.Write([]byte(PrivateKey))
    hasher.Write([]byte(DataBase64))
    hasher.Write([]byte(PrivateKey))

    SignBase64 := b64.StdEncoding.EncodeToString(hasher.Sum(nil))
    fmt.Println(SignBase64)

    form := url.Values{
        "data":      {DataBase64},
        "signature": {SignBase64},
    }

    body := bytes.NewBufferString(form.Encode())
    rsp, err := http.Post(LiqpayURL + apiUrl, "application/x-www-form-urlencoded", body)
    if err != nil {
        panic(err)
    }
    defer rsp.Body.Close()
    body_byte, err := ioutil.ReadAll(rsp.Body)
    if err != nil {
        panic(err)
    }
    fmt.Println(string(body_byte))

}