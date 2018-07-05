package main

import (
    "encoding/json"
    "fmt"
    "bytes"
    "io/ioutil"
    "net/http"
    "net/url"
    "crypto/sha1"
    "html/template"
    "os"
)
import b64 "encoding/base64"


var PublicKey  = ""
var PrivateKey = ""
var LiqpayURL  = "https://www.liqpay.ua/api/"


type DataSignature struct {
    Data string
    Signature string 
}


func Init(PubKey string, PrivKey string) {
    PublicKey  = PubKey
    PrivateKey = PrivKey
}


func Api(apiUrl string, Data map[string]interface{}) {
    
    DataBytes, _ := json.Marshal(Data)
    
    DataString := string(DataBytes)
    fmt.Println("JSON:",DataString)

    DataBase64 := MakeData(DataString)
    fmt.Println("Data:",DataBase64)

    SignBase64 := MakeSignature(DataBase64)
    fmt.Println("Signature:",SignBase64)

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
    fmt.Println("Liqpay response:",string(body_byte))

}


func Form(Data map[string]interface{}) {
    
    DataBytes, _ := json.Marshal(Data)
    
    DataString := string(DataBytes)
    fmt.Println("JSON:",DataString)

    DataBase64 := MakeData(DataString)
    fmt.Println("Data:",DataBase64)

    SignBase64 := MakeSignature(DataBase64)
    fmt.Println("Signature:",SignBase64)

    t, _ := template.ParseFiles("liqpay_form.html")
    t.ExecuteTemplate(os.Stdout, "liqpay_form.html", DataSignature{
        Data: DataBase64,
        Signature: SignBase64,
    })

}


func MakeData (Data string) (string) {
    return b64.StdEncoding.EncodeToString([]byte(Data))
}


func MakeSignature (DataBase64 string) (string) {
    hasher := sha1.New()
    hasher.Write([]byte(PrivateKey))
    hasher.Write([]byte(DataBase64))
    hasher.Write([]byte(PrivateKey))
    SignBase64 := b64.StdEncoding.EncodeToString(hasher.Sum(nil))
    return SignBase64
}