package main

import "strings"

type BtpAddress string

func (a BtpAddress) Protocol() string {
	s := string(a)
	if i := strings.Index(s, "://"); i > 0 {
		return s[:i]
	}
	return ""
}
func (a BtpAddress) NetworkAddress() string {
	if a.Protocol() != "" {
		ss := strings.Split(string(a), "/")
		if len(ss) > 2 {
			return ss[2]
		}
	}
	return ""
}
func (a BtpAddress) network() (string, string) {
	if s := a.NetworkAddress(); s != "" {
		ss := strings.Split(s, ".")
		if len(ss) > 1 {
			return ss[0], ss[1]
		} else {
			return "", ss[0]
		}
	}
	return "", ""
}
func (a BtpAddress) BlockChain() string {
	_, v := a.network()
	return v
}
func (a BtpAddress) NetworkID() string {
	n, _ := a.network()
	return n
}
func (a BtpAddress) ContractAddress() string {
	if a.Protocol() != "" {
		ss := strings.Split(string(a), "/")
		if len(ss) > 3 {
			return ss[3]
		}
	}
	return ""
}

func (a BtpAddress) String() string {
	return string(a)
}

func (a *BtpAddress) Set(v string) error {
	*a = BtpAddress(v)
	return nil
}

func (a BtpAddress) Type() string {
	return "BtpAddress"
}
