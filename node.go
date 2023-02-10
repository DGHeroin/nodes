package nodes

import (
    "encoding/json"
    "errors"
    "github.com/hashicorp/memberlist"
    "io"
    "net"
    "strconv"
    "time"
)

type (
    Node struct {
        opt     Option
        msgChan chan Message
        list    *memberlist.Memberlist
    }
    Option struct {
        Name    string
        Address string
        Members []string
    }
    Message struct {
        Type     int
        NodeName string
        Payload  []byte
    }
    Opts func(o *Option)
)

const (
    MsgJoin = iota
    MsgLeave
    MsgNotify
)

var (
    ErrNotExist = errors.New("node not exists")
)

func (n *Node) emit(msg Message) {
    go func() {
        n.msgChan <- msg
    }()
}
func (n *Node) NodeMeta(limit int) []byte                                                    { return nil }
func (n *Node) GetBroadcasts(overhead, limit int) [][]byte                                   { return nil }
func (n *Node) LocalState(join bool) []byte                                                  { return nil }
func (n *Node) MergeRemoteState(buf []byte, join bool)                                       {}
func (n *Node) NotifyUpdate(node *memberlist.Node)                                           {}
func (n *Node) NotifyConflict(existing, other *memberlist.Node)                              {}
func (n *Node) NotifyMerge(peers []*memberlist.Node) error                                   { return nil }
func (n *Node) AckPayload() []byte                                                           { return nil }
func (n *Node) NotifyPingComplete(other *memberlist.Node, rtt time.Duration, payload []byte) {}
func (n *Node) NotifyAlive(peer *memberlist.Node) error                                      { return nil }
func (n *Node) NotifyJoin(node *memberlist.Node) {
    n.emit(Message{
        Type:     MsgJoin,
        NodeName: node.Name,
    })
}
func (n *Node) NotifyLeave(node *memberlist.Node) {
    n.emit(Message{
        Type:     MsgLeave,
        NodeName: node.Name,
    })
}
func (n *Node) NotifyMsg(bytes []byte) {
    msg := struct {
        Sender  string `json:"sender,omitempty"`
        Payload []byte `json:"payload,omitempty"`
    }{}
    err := json.Unmarshal(bytes, &msg)
    if err != nil {
        return
    }
    n.emit(Message{
        Type:     MsgNotify,
        NodeName: msg.Sender,
        Payload:  msg.Payload,
    })
}
func (n *Node) Start() error {
    // 配置
    config := memberlist.DefaultLocalConfig()
    config.Name = n.opt.Name
    ip, port, err := net.SplitHostPort(n.opt.Address)
    if err != nil {
        return err
    }
    config.BindAddr = ip
    config.BindPort, _ = strconv.Atoi(port)

    // 配置 delegate
    config.Delegate = n
    config.Events = n
    config.Conflict = n
    config.Merge = n
    config.Ping = n
    config.Alive = n
    config.LogOutput = io.Discard
    // 启动
    list, err := memberlist.Create(config)
    if err != nil {
        return err
    }
    n.list = list

    if len(n.opt.Members) > 0 {
        for {
            _, err := list.Join(n.opt.Members)
            if err != nil {
                time.Sleep(time.Second)
                continue
            }
            break
        }
    }
    return nil
}
func (n *Node) MessageChan() <-chan Message { return n.msgChan }
func (n *Node) BroadcastJson(p interface{}) error {
    data, err := json.Marshal(p)
    if err != nil {
        return err
    }
    n.Broadcast(data)
    return nil
}
func (n *Node) Broadcast(payload []byte) {
    for _, node := range n.list.Members() {
        _ = n.SendTo(node.Name, payload)
    }
}
func (n *Node) SendToJson(nodeName string, p interface{}) error {
    data, err := json.Marshal(p)
    if err != nil {
        return err
    }
    return n.SendTo(nodeName, data)
}
func (n *Node) SendTo(nodeName string, payload []byte) error {
    if nodeName == n.GetMyNodeName() {
        return nil
    }
    targetNode := n.getNode(nodeName)
    if targetNode == nil {
        return ErrNotExist
    }
    msg := struct {
        Sender  string `json:"sender,omitempty"`
        Payload []byte `json:"payload,omitempty"`
    }{
        Sender:  n.GetMyNodeName(),
        Payload: payload,
    }
    data, _ := json.Marshal(msg)
    return n.list.SendReliable(targetNode, data)
}
func (n *Node) getNode(nodeName string) *memberlist.Node {
    for _, node := range n.list.Members() {
        if node.Name == nodeName {
            return node
        }
    }
    return nil
}
func (n *Node) GetMyNodeName() string           { return n.list.LocalNode().Name }
func (m *Message) BindJson(p interface{}) error { return json.Unmarshal(m.Payload, p) }
func NewNode(opts ...Opts) *Node {
    opt := defaultOption()
    FromEnv()(&opt)
    for _, o := range opts {
        o(&opt)
    }
    node := &Node{
        opt:     opt,
        msgChan: make(chan Message, 10),
    }
    return node
}

type simpleBroadcast []byte

func (b simpleBroadcast) Message() []byte                       { return []byte(b) }
func (b simpleBroadcast) Invalidates(memberlist.Broadcast) bool { return false }
func (b simpleBroadcast) Finished()                             {}
