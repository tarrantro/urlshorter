package main

import (
	"fmt"
	"golang.org/x/net/idna"
	"hash/fnv"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Domain struct {
	Host string
	ID   int
}

func FNV32a(text string) uint32 {
	algorithm := fnv.New32a()
	algorithm.Write([]byte(text))
	return algorithm.Sum32()
}

func transformDomain(domain string, ports ...string) ([]Domain, error) {
	port := Getenv("HOST_PORT", "8080")
	if len(ports) > 0 {
		port = ports[0]
	}
	rs := make([]Domain, 0)
	if net.ParseIP(domain).To4() == nil {
		if _, err := idna.Lookup.ToASCII(domain); err != nil {
			return rs, fmt.Errorf("%v is not an IPv4 address", domain)
		}
		ips, err := net.LookupHost(domain)
		if err != nil {
			return rs, fmt.Errorf("could not find records for domain %s", domain)
		}
		for _, ip := range ips {
			i := fmt.Sprintf("http://%s:%s", ip, port)
			rs = append(rs, Domain{Host: i, ID: int(FNV32a(i) % 1024)})
		}

	} else {
		i := fmt.Sprintf("http://%s:%s", domain, port)
		rs = append(rs, Domain{Host: i, ID: int(FNV32a(i) % 1024)})
	}
	return rs, nil
}

func node() (*Node, error) {
	var err error
	hostname := Getenv("HOST_ADDRESS", "")
	if hostname == "" {
		hostname, err = os.Hostname()
		if err != nil {
			return nil, err
		}
	}
	nodes := strings.Split(string(Getenv("NODE_LIST", "")), ",")

	hList, err := transformDomain(hostname)
	if err != nil {
		return nil, err
	}

	for _, node := range nodes {
		if node == "" {
			continue
		}
		t := strings.Split(string(node), ":")
		port := "8080"
		if len(t) > 1 {
			port = t[1]
		}
		domain := t[0]
		d, err := transformDomain(domain, port)
		if err != nil {
			return nil, err
		}
		hList = append(hList, d...)
	}

	sort.Slice(hList, func(i, j int) bool {
		return hList[i].ID < hList[j].ID
	})

	id := int(FNV32a(hostname) % 1024)
	if len(hList) > 1 {
		for _, i := range hList {
			if i.ID == id {
				h := i.Host
				if h != hostname {
					resp, err := http.Get(fmt.Sprintf("%s/node", h))
					defer resp.Body.Close()
					if err != nil {
						return nil, fmt.Errorf("GET id from node %s error: %v", h, err)
					}

					data, err := io.ReadAll(resp.Body)
					if err != nil {
						return nil, fmt.Errorf("Read body from node %s :error %v", h, err)
					}
					tid, err := strconv.Atoi(string(data))
					if err != nil {
						return nil, fmt.Errorf("Read data from node %s :error %v", h, err)
					}
					if tid == id {
						id += 1
					}
				}
			}
		}
	}
	fmt.Printf("Start new node %d\n", id)
	return NewNode(id)
}
