package main

import (
    "fmt"
    "github.com/DGHeroin/nodes"
    "log"
    "time"
)

func main() {
    log.SetFlags(log.LstdFlags | log.Lshortfile)

    n := nodes.NewNode()
    err := n.Start()
    if err != nil {
        log.Println(err)
        return
    }
    go func() {
        for {
            time.Sleep(time.Second * 5)
            n.Broadcast([]byte("我很好"))
        }
    }()
    for msg := range n.MessageChan() {
        switch msg.Type {
        case nodes.MsgJoin:
            log.Println("[加入]", msg.NodeName)
            _ = n.SendTo(msg.NodeName, []byte(fmt.Sprintf("%s: 你来了呀", n.GetMyNodeName())))
        case nodes.MsgLeave:
            log.Println("[离开]", msg.NodeName)
        case nodes.MsgNotify:
            log.Println("[通知]", string(msg.Payload), msg.NodeName)
        }
    }
}
