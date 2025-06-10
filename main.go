package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func getLocalIP() (net.IP, error) {
	// デフォルトルートのインターフェースを取得
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}

func main() {
	// コマンドライン引数の設定
	sport := flag.Int("sport", 0, "Source port number (0-65535). Optional. Random if omitted.")
	count := flag.Int("count", 1, "Number of packets to send")
	dst := flag.String("dst", "", "Destination IP address or hostname")
	dport := flag.Int("dport", 0, "Destination port")
	flag.Parse()

	// 必須パラメータのチェック
	if *dst == "" || *dport == 0 {
		fmt.Println("Error: --dst and --dport are required")
		return
	}

	// ローカルIPアドレスの取得
	localIP, err := getLocalIP()
	if err != nil {
		fmt.Printf("Error getting local IP: %v\n", err)
		return
	}
	fmt.Printf("Using local IP: %s\n", localIP)

	// raw socketの作成
	conn, err := net.ListenPacket("ip4:tcp", "0.0.0.0")
	if err != nil {
		fmt.Printf("Error creating raw socket: %v\n", err)
		return
	}
	defer conn.Close()

	// ランダムシードの初期化
	rand.Seed(time.Now().UnixNano())

	// パケットの送信
	for i := 0; i < *count; i++ {
		// 送信元ポートの設定
		srcPort := *sport
		if srcPort == 0 {
			srcPort = rand.Intn(65535-1024) + 1024
		}

		// IPレイヤーの作成
		ip := &layers.IPv4{
			SrcIP:    localIP,
			DstIP:    net.ParseIP(*dst),
			Protocol: layers.IPProtocolTCP,
		}

		// TCPレイヤーの作成
		tcp := &layers.TCP{
			SrcPort: layers.TCPPort(srcPort),
			DstPort: layers.TCPPort(*dport),
			SYN:     true,
			Seq:     100,
			Window:  14600,
			Options: []layers.TCPOption{},
		}

		// TCPレイヤーにネットワークレイヤーを設定
		tcp.SetNetworkLayerForChecksum(ip)

		// バッファの作成
		buffer := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{
			FixLengths:       true,
			ComputeChecksums: true,
		}

		// IP+TCPをまとめてシリアライズ
		if err := gopacket.SerializeLayers(buffer, opts, ip, tcp); err != nil {
			fmt.Printf("Error serializing packet: %v\n", err)
			continue
		}

		// パケットの送信
		fmt.Printf("Sending to: %s:%d ... ", *dst, *dport)

		// パケットを送信
		_, err := conn.WriteTo(buffer.Bytes(), &net.IPAddr{IP: net.ParseIP(*dst)})
		if err != nil {
			fmt.Printf("Error sending packet: %v\n", err)
			continue
		}

		// レスポンスの受信を試みる
		conn.SetReadDeadline(time.Now().Add(time.Second))
		resp := make([]byte, 1024)
		n, _, err := conn.ReadFrom(resp)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				fmt.Println("No response received (timeout)")
			} else {
				fmt.Printf("Error receiving response: %v\n", err)
			}
			continue
		}

		// レスポンスの解析
		packet := gopacket.NewPacket(resp[:n], layers.LayerTypeTCP, gopacket.Default)
		if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
			tcp, _ := tcpLayer.(*layers.TCP)
			if tcp.SYN && tcp.ACK {
				fmt.Println("Received SYN-ACK")
			} else {
				fmt.Println("Received unexpected response")
			}
		} else {
			fmt.Println("Received non-TCP response")
		}
	}
}
