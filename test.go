package main

func main() {

    Init("public_key", "private_key")

    Api("request", map[string]interface{}{
        "action": "agent_info_merchant",
        "version": 3,
        "public_key": PublicKey,
        "merchant_public_key": PublicKey,
        "split_rules": []interface{}{
            map[string]interface{}{
                "public_key": PublicKey,
                "amount": 0.1,
            },
        },
    })   

}
