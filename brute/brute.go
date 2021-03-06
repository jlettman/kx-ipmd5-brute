package brute

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net"
	"os"
)

// Hashes represents a constant-time unique lookup table for checking whether a hash is in the pool of those to break.
type Hashes map[string]struct{}

// HashResult represents the result of MD5-hashing an IPv4 address.
type HashResult struct {
	Hash string
	IP   net.IP
}

// incIP returns an IPv4 address one greater than the provided IPv4 address.
func incIP(ip net.IP) net.IP {
	i := make(net.IP, 4)
	copy(i, ip)

	for j := len(i) - 1; j >= 0; j-- {
		i[j]++

		if i[j] > 0 {
			break
		}
	}

	return i
}

// HashIP returns a HashResult pointer with the IP and hex-encoded MD5 value.
func HashIP(ip net.IP) *HashResult {
	hasher := md5.New()
	hasher.Write([]byte(ip.String()))

	i := make(net.IP, 4)
	copy(i, ip)

	return &HashResult{
		Hash: hex.EncodeToString(hasher.Sum(nil)),
		IP:   i,
	}
}

// BruteIPNet bruteforces an IPv4 subnet and pushes results matching the hashes sought to the provided channel.
func BruteIPNet(ipnet *net.IPNet, hashes Hashes, channel chan<- *HashResult) {
	// iterate over the range of IPv4 addresses in the provided IPv4 subnet,
	for ip := ipnet.IP; ipnet.Contains(ip); ip = incIP(ip) {
		result := HashIP(ip) // hash the current IPv4 address

		// check if the hash exists in the table,
		if _, ok := hashes[result.Hash]; ok {
			// output the result,
			fmt.Printf("match found! %s = %s\n", result.Hash, result.IP.String())
			channel <- result
		}
	}
}

func BruteCIDR(cidr string, hashes Hashes, channel chan<- *HashResult) error {
	_, ipnet, err := net.ParseCIDR(cidr)

	if err != nil {
		return err
	}

	BruteIPNet(ipnet, hashes, channel)
	return nil
}

func BruteIPNetWorker(id int, hashes Hashes, jobs <-chan *net.IPNet, channel chan<- *HashResult) {
	jid := 0

	for ipnet := range jobs {
		jid++
		fmt.Printf("worker #%v starting job #%v for IPNet %v\n", id, jid, ipnet)
		BruteIPNet(ipnet, hashes, channel)
		fmt.Printf("worker #%v finished job #%v for IPNet %v\n", id, jid, ipnet)
	}
}

func FileHashResultWrite(file *os.File, channel chan *HashResult) {
	for result := range channel {
		fmt.Printf("writing %s=%v\n", result.Hash, result.IP)
		fmt.Fprintf(file, "%s=%s\n", result.Hash, result.IP.String())
	}
}

func FileHashesRead(file *os.File) Hashes {
	hashes := make(Hashes)

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		hash := scanner.Text()
		fmt.Printf("loading hash: %s\n", hash)
		hashes[hash] = struct{}{}
	}

	return hashes
}
