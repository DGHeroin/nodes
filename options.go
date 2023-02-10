package nodes

import (
    "math/rand"
    "os"
    "strconv"
    "strings"
)

func defaultOption() Option {
    return Option{
        Name:    "nodes-" + strconv.Itoa(rand.Intn(9999)+1000),
        Address: "127.0.0.1:0",
    }
}
func FromEnv() Opts {
    return func(o *Option) {
        o.Name = os.Getenv("node_name")
        o.Address = os.Getenv("node_address")
        if str := strings.TrimSpace(os.Getenv("node_members")); str != "" {
            o.Members = strings.Split(str, ",")
        }
        if o.Address == "" {
            o.Address = "127.0.0.1:0"
        }

        if o.Name == "" {
            hostname, _ := os.Hostname()
            o.Name = hostname + "-" + o.Address + "#" + strconv.Itoa(rand.Intn(9999)+10000)
        }
    }
}
