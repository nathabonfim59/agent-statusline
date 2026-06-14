package proxy

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nathabonfim59/agent-statusline/harness"
)

type ProxyServer struct {
	config    harness.ProxyConfig
	filter    *DomainFilter
	listener  net.Listener
	dataLn    net.Listener
	dataPort  int
	caCert    *x509.Certificate
	caKey     *rsa.PrivateKey
	certCache map[string]*tls.Certificate
	certMu    sync.Mutex
	done      chan struct{}
}

func Start(config harness.ProxyConfig) (*ProxyServer, error) {
	certPEM, keyPEM, err := LoadOrGenerateCA()
	if err != nil {
		return nil, fmt.Errorf("load CA: %w", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	caCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CA cert: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	caKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CA key: %w", err)
	}

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, err
	}

	dataLn, err := net.Listen("tcp", ":0")
	if err != nil {
		ln.Close()
		return nil, err
	}

	s := &ProxyServer{
		config:    config,
		filter:    NewDomainFilter(config.Domains),
		listener:  ln,
		dataLn:    dataLn,
		dataPort:  dataLn.Addr().(*net.TCPAddr).Port,
		caCert:    caCert,
		caKey:     caKey,
		certCache: make(map[string]*tls.Certificate),
		done:      make(chan struct{}),
	}

	go s.serveData()
	go s.serve()

	return s, nil
}

func (s *ProxyServer) Port() int     { return s.listener.Addr().(*net.TCPAddr).Port }
func (s *ProxyServer) DataPort() int { return s.dataPort }

func (s *ProxyServer) Stop() {
	close(s.done)
	s.listener.Close()
	s.dataLn.Close()
}

func (s *ProxyServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
			}
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *ProxyServer) handleConn(client net.Conn) {
	defer client.Close()

	br := bufio.NewReader(client)
	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}

	if req.Method == "CONNECT" {
		s.handleConnect(client, br, req)
		return
	}

	s.handleHTTP(client, br, req)
}

func (s *ProxyServer) handleConnect(client net.Conn, br *bufio.Reader, req *http.Request) {
	hostPort := req.URL.Host
	host := hostPort
	if h, _, err := net.SplitHostPort(hostPort); err == nil {
		host = h
	}

	if !s.filter.ShouldIntercept(host) {
		s.tunnel(client, hostPort)
		return
	}

	client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	tlsCert := s.getCert(host)
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{*tlsCert}}
	tlsClient := tls.Server(client, tlsCfg)
	defer tlsClient.Close()

	tlsBR := bufio.NewReader(tlsClient)
	for {
		req, err := http.ReadRequest(tlsBR)
		if err != nil {
			return
		}

		server, err := tls.Dial("tcp", hostPort, &tls.Config{})
		if err != nil {
			return
		}

		if err := req.Write(server); err != nil {
			server.Close()
			return
		}

		resp, err := http.ReadResponse(bufio.NewReader(server), req)
		if err != nil {
			server.Close()
			return
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		ct := resp.Header.Get("Content-Type")
		if s.config.Collector != nil {
			s.config.Collector.HandleResponse(host, req.URL.Path, ct, body)
		}

		resp.Body = io.NopCloser(strings.NewReader(string(body)))
		resp.Write(tlsClient)
		server.Close()

		if !resp.Close && !req.Close {
			continue
		}
		return
	}
}

func (s *ProxyServer) tunnel(client net.Conn, hostPort string) {
	server, err := net.DialTimeout("tcp", hostPort, 10*time.Second)
	if err != nil {
		client.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer server.Close()

	client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(server, client)
		server.(*net.TCPConn).CloseWrite()
	}()
	go func() {
		defer wg.Done()
		io.Copy(client, server)
		client.(*net.TCPConn).CloseWrite()
	}()
	wg.Wait()
}

func (s *ProxyServer) handleHTTP(client net.Conn, br *bufio.Reader, req *http.Request) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		client.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if s.config.Collector != nil && s.filter.ShouldIntercept(req.Host) {
		ct := resp.Header.Get("Content-Type")
		s.config.Collector.HandleResponse(req.Host, req.URL.Path, ct, body)
	}

	resp.Body = io.NopCloser(strings.NewReader(string(body)))
	resp.Write(client)
}

func (s *ProxyServer) getCert(host string) *tls.Certificate {
	s.certMu.Lock()
	defer s.certMu.Unlock()

	if c, ok := s.certCache[host]; ok {
		return c
	}

	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: host},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{host},
	}

	der, _ := x509.CreateCertificate(rand.Reader, tmpl, s.caCert, &priv.PublicKey, s.caKey)
	cert := &tls.Certificate{
		Certificate: [][]byte{der, s.caCert.Raw},
		PrivateKey:  priv,
	}
	s.certCache[host] = cert
	return cert
}

func (s *ProxyServer) serveData() {
	mux := http.NewServeMux()
	mux.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		if s.config.Collector == nil {
			w.WriteHeader(404)
			return
		}
		data := s.config.Collector.GetData()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	})
	http.Serve(s.dataLn, mux)
}