package collect

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// fetchNewsgroups connects to the local NNTP server and fetches group info
func fetchNewsgroups(ctx context.Context) ([]NewsgroupInfo, error) {
	conn, err := net.DialTimeout("tcp", "localhost:119", 1*time.Second)
	if err != nil {
		return nil, fmt.Errorf("nntp connect: %w", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(1500 * time.Millisecond))

	reader := bufio.NewReader(conn)

	// Read initial greeting
	if _, err := reader.ReadString('\n'); err != nil {
		return nil, fmt.Errorf("nntp greeting: %w", err)
	}

	// MODE-READER (required before GROUP/XOVER on inn2)
	fmt.Fprintf(conn, "MODE READER\r\n")
	if _, err := reader.ReadString('\n'); err != nil {
		return nil, fmt.Errorf("nntp mode-reader: %w", err)
	}

	// LIST newsgroups
	fmt.Fprintf(conn, "LIST\r\n")
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("nntp list: %w", err)
	}
	// 215 = success
	if !strings.HasPrefix(line, "215") {
		return nil, fmt.Errorf("nntp list failed: %s", strings.TrimSpace(line))
	}

	var groups []NewsgroupInfo
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if line == "." {
			break
		}
		// Format: group high low [post|y|n] [description]
		parts := strings.SplitN(line, " ", 4)
		if len(parts) < 3 {
			continue
		}
		high, _ := strconv.Atoi(parts[1])
		low, _ := strconv.Atoi(parts[2])

		groups = append(groups, NewsgroupInfo{
			Name:     parts[0],
			NewCount: high - low + 1,
			Low:      low,
		})
	}

	return groups, nil
}
