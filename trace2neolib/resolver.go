package trace2neolib

import (
	"fmt"
	"net"
	"strings"
)

type ResolvedAddr struct {
	Addr  string
	Names []string
}

type Asset struct {
	ShortName string
	Label     string
	Name      string
	IPAddr    string
}

func ResolveAddr(addr string) (*ResolvedAddr, error) {
	names, err := net.LookupAddr(addr)
	if err != nil {
		return nil, err
	}

	return &ResolvedAddr{
		Addr:  addr,
		Names: names,
	}, nil
}

func stripCharacters(str, chrs string) string {
	return strings.Map(func(r rune) rune {
		if strings.IndexRune(chrs, r) < 0 {
			return r
		}
		return -1
	}, str)
}

func ResolvedAddrToAsset(resolved *ResolvedAddr, ip string, iteration int) []*Asset {
	var assets []*Asset
	if resolved != nil {
		if len(resolved.Names) > 0 {
			for _, name := range resolved.Names {
				assets = append(assets, &Asset{
					Name:      name,
					IPAddr:    resolved.Addr,
					ShortName: fmt.Sprintf("var%s%d", stripCharacters(name, "`~!@#$%^&*()-_=+[]{]}\t\\|'\";:,<.>/?\n `"), iteration),
					Label:     "Unknown",
				})
			}
			return assets
		}
	}

	strippedIP := stripCharacters(ip, ".:[]")
	assets = append(assets, &Asset{
		Name:      ip,
		IPAddr:    ip,
		ShortName: fmt.Sprintf("var%s", strippedIP),
		Label:     "Unknown",
	})

	return assets
}
