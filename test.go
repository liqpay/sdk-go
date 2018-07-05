package main

import ("fmt")

func main() {

    fmt.Println("Run Init test...")

    Init("i42883299811", "T1cBpWrXqjPFiItoQiMyXTwK88mv9baUEqviL47g")

    fmt.Println("PublicKey =",PublicKey)
    fmt.Println("PrivateKey =",PrivateKey)

    fmt.Println("Run Api test...")

    Api("request", map[string]interface{}{
        "action": "pay",
        "version": 3,
        "public_key": PublicKey,
        "amount": 1,
        "currency": "UAH",
        "description": "Test payment",
        "order_id": "order_id_1",
        "split_rules": []interface{}{
            map[string]interface{}{
                "public_key": PublicKey,
                "amount": 0.1,
            },
        },
    })

    fmt.Println("Run Form test...")

    Form(map[string]interface{}{
        "action": "pay",
        "version": 3,
        "public_key": PublicKey,
        "amount": 1,
        "currency": "UAH",
        "description": "Test payment",
        "order_id": "order_id_1",
    })   

}