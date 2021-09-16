package main

import (
  "fmt"
  "encoding/binary"
  "errors"
  "net"
  "regexp"
  "strconv"
  "os"
)

var net_regex *regexp.Regexp
var ip_or_net_regex *regexp.Regexp

func init() {
  net_regex = regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+)\/(\d{1,2})$`)
  ip_or_net_regex = regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+)(?:\/(\d{1,2}))?$`)
}

func Ip2long(ipAddr string) (uint32, error) {
	ip := net.ParseIP(ipAddr)
	if ip == nil {
		return 0, errors.New("wrong ipAddr format")
	}
	ip = ip.To4()
	return binary.BigEndian.Uint32(ip), nil
}

func Long2ip(ipLong uint32) string {
	ipByte := make([]byte, 4)
	binary.BigEndian.PutUint32(ipByte, ipLong)
	ip := net.IP(ipByte)
	return ip.String()
}

func Mask(masklen uint8) uint32 {
  return uint32(0xFFFFFFFF) << (32-masklen)
}

type Net struct {
  Addr uint32
  Masklen uint8
}

func (net *Net) Contains(child *Net) bool {
  if (net.Addr & Mask(net.Masklen)) == (child.Addr & Mask(net.Masklen)) && net.Masklen <= child.Masklen {
    return true
  } else {
    return false
  }
}

func (net *Net) IsValid() bool {

//fmt.Println("IsValid:", Long2ip(net.Addr)+"/"+fmt.Sprint(net.Masklen))
//fmt.Printf("\t net: %032b\n", net.Addr)
//fmt.Printf("\tmask: %032b\n", Mask(net.Masklen))

  if (net.Addr & Mask(net.Masklen)) == net.Addr {
    return true
  } else {
    return false
  }
}

func (net *Net) String() string {
if !net.IsValid() {
  panic("Invalid network for String: " + Long2ip(net.Addr)+"/"+fmt.Sprint(net.Masklen))
}
  return Long2ip(net.Addr)+"/"+fmt.Sprint(net.Masklen)
}

func (net *Net) Split() []*Net {
  if net.Masklen >= 32 {
    return nil
  }

  var next_masklen uint8 = net.Masklen + 1

  ret := make([]*Net, 2)

  ret[0] = &Net{net.Addr, next_masklen}
  ret[1] = &Net{net.Addr, next_masklen}

  var bit uint32 = (1 << (32 - next_masklen))
  ret[1].Addr = ret[1].Addr | bit

  return ret
}

func (net *Net) Equals(check_net *Net) bool {
  return net.Addr == check_net.Addr && net.Masklen == check_net.Masklen
}

func (net *Net) Exclude(exclude_nets []*Net) {
  contains := false
  for _, exclude_net := range exclude_nets {
    if net.Equals(exclude_net) {
      return
    }
    if net.Contains(exclude_net) {
      contains = true
      break
    }
  }

  if contains {
    nets := net.Split()
    for _, part := range nets {
      part.Exclude(exclude_nets)
    }
  } else {
    fmt.Println(net)
  }
}

func main() {

  var err error

  if len(os.Args) < 3 {
    fmt.Fprintln(os.Stderr, "USAGE: "+os.Args[0]+" n.n.n.n/n x.x.x.x/x [...]\n\tn.n.n.n/n - network to use\n\tx.x.x.x/x - networks to exclude, omit /x for /32")
    os.Exit(1)
  }

  start_net_m := net_regex.FindStringSubmatch(os.Args[1])
  if len(start_net_m) != 3 {
    fmt.Fprintln(os.Stderr, "Bad network: "+os.Args[1])
    os.Exit(1)
  }

  var start_addr uint32
  start_addr, err = Ip2long(start_net_m[1])
  if err != nil {
    fmt.Fprintln(os.Stderr, "Bad network: "+os.Args[1])
    os.Exit(1)
  }

  var start_masklen uint64
  start_masklen, err = strconv.ParseUint(start_net_m[2], 10, 8)
  if err != nil {
    fmt.Fprintln(os.Stderr, "Bad network: "+os.Args[1])
    os.Exit(1)
  }

  if start_masklen > 32 {
    fmt.Fprintln(os.Stderr, "Bad network: "+os.Args[1])
    os.Exit(1)
  }

  start_net := &Net{start_addr, uint8(start_masklen)}

  if !start_net.IsValid() {
    fmt.Fprintln(os.Stderr, "Invalid network/mask: "+os.Args[1])
    os.Exit(1)
  }

  exclude_nets := make([]*Net, 0)

  for _, arg := range os.Args[2:] {

    exclude_net_m := ip_or_net_regex.FindStringSubmatch(arg)
    if len(exclude_net_m) != 3 {
      fmt.Fprintln(os.Stderr, "Bad network: "+arg)
      os.Exit(1)
    }

    var exclude_addr uint32
    exclude_addr, err = Ip2long(exclude_net_m[1])
    if err != nil {
      fmt.Fprintln(os.Stderr, "Bad network: "+arg)
      os.Exit(1)
    }

    var exclude_masklen uint64

    if exclude_net_m[2] != "" {
      exclude_masklen, err = strconv.ParseUint(exclude_net_m[2], 10, 8)
      if err != nil {
        fmt.Fprintln(os.Stderr, "Bad network: "+arg)
        os.Exit(1)
      }
    } else {
      exclude_masklen = 32
    }

    if exclude_masklen > 32 {
      fmt.Fprintln(os.Stderr, "Bad network: "+arg)
      os.Exit(1)
    }

    exclude_net := &Net{exclude_addr, uint8(exclude_masklen)}

    if !exclude_net.IsValid() {
      fmt.Fprintln(os.Stderr, "Invalid network/mask: "+arg)
      os.Exit(1)
    }

    if exclude_net.Contains(start_net) {
      return
    }

    exclude_nets = append(exclude_nets, exclude_net)
  }

  start_net.Exclude(exclude_nets)

}
