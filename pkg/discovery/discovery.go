package discovery

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type DeviceInfo struct {
	IP           string `json:"ip"`
	Name         string `json:"name"`
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	ESCLURL      string `json:"escl_url"`
	IPPURL       string `json:"ipp_url"`
}

// Discover probes local subnets for eSCL / IPP enabled printers
func Discover(timeout time.Duration) ([]DeviceInfo, error) {
	subnets := getLocalSubnetPrefixes()
	if len(subnets) == 0 {
		subnets = []string{"192.168.31.", "192.168.1.", "192.168.0."}
	}

	var results []DeviceInfo
	var mu sync.Mutex
	var wg sync.WaitGroup

	client := &http.Client{
		Timeout: 500 * time.Millisecond,
	}

	for _, prefix := range subnets {
		for i := 1; i < 255; i++ {
			ip := fmt.Sprintf("%s%d", prefix, i)
			wg.Add(1)
			go func(targetIP string) {
				defer wg.Done()
				dev, ok := probeIP(client, targetIP)
				if ok {
					mu.Lock()
					results = append(results, dev)
					mu.Unlock()
				}
			}(ip)
		}
	}

	wg.Wait()
	return results, nil
}

// QuickFirst finds the first reachable printer IP
func QuickFirst(timeout time.Duration) (*DeviceInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	subnets := getLocalSubnetPrefixes()
	if len(subnets) == 0 {
		subnets = []string{"192.168.31.", "192.168.1.", "192.168.0."}
	}

	foundChan := make(chan DeviceInfo, 1)
	client := &http.Client{
		Timeout: 600 * time.Millisecond,
	}

	var wg sync.WaitGroup
	for _, prefix := range subnets {
		for i := 1; i < 255; i++ {
			ip := fmt.Sprintf("%s%d", prefix, i)
			wg.Add(1)
			go func(targetIP string) {
				defer wg.Done()
				select {
				case <-ctx.Done():
					return
				default:
					dev, ok := probeIP(client, targetIP)
					if ok {
						select {
						case foundChan <- dev:
							cancel()
						default:
						}
					}
				}
			}(ip)
		}
	}

	go func() {
		wg.Wait()
		close(foundChan)
	}()

	select {
	case dev, ok := <-foundChan:
		if ok {
			return &dev, nil
		}
		return nil, fmt.Errorf("no printer discovered on local network")
	case <-ctx.Done():
		return nil, fmt.Errorf("discovery timeout")
	}
}

func probeIP(client *http.Client, ip string) (DeviceInfo, bool) {
	url := fmt.Sprintf("http://%s/eSCL/ScannerCapabilities", ip)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return DeviceInfo{}, false
	}
	req.Header.Set("User-Agent", "printctl/0.0.1-alpha")

	resp, err := client.Do(req)
	if err != nil {
		return DeviceInfo{}, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return DeviceInfo{}, false
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return DeviceInfo{}, false
	}

	content := string(body)
	if !strings.Contains(content, "ScannerCapabilities") {
		return DeviceInfo{}, false
	}

	name := "Network Printer"
	if strings.Contains(content, "<pwg:MakeAndModel>") {
		parts := strings.Split(content, "<pwg:MakeAndModel>")
		if len(parts) > 1 {
			end := strings.Index(parts[1], "</pwg:MakeAndModel>")
			if end > 0 {
				name = parts[1][:end]
			}
		}
	}

	return DeviceInfo{
		IP:           ip,
		Name:         name,
		Manufacturer: "Standard",
		Model:        name,
		ESCLURL:      fmt.Sprintf("http://%s/eSCL", ip),
		IPPURL:       fmt.Sprintf("http://%s:631/ipp/print", ip),
	}, true
}

func getLocalSubnetPrefixes() []string {
	var prefixes []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return prefixes
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil {
				continue
			}
			ip := ipNet.IP.To4()
			if ip[0] == 192 && ip[1] == 168 {
				prefixes = append(prefixes, fmt.Sprintf("%d.%d.%d.", ip[0], ip[1], ip[2]))
			} else if ip[0] == 10 {
				prefixes = append(prefixes, fmt.Sprintf("%d.%d.%d.", ip[0], ip[1], ip[2]))
			} else if ip[0] == 172 && (ip[1] >= 16 && ip[1] <= 31) {
				prefixes = append(prefixes, fmt.Sprintf("%d.%d.%d.", ip[0], ip[1], ip[2]))
			}
		}
	}
	return prefixes
}
