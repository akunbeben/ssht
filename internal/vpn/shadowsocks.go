package vpn

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

// ShadowSocks AEAD Dialer implementation without external dependencies
// Supports: aes-128-gcm, aes-256-gcm, chacha20-ietf-poly1305

type ssConfig struct {
	Method   string
	Password string
	Server   string
}

func parseSSUri(uri string) (*ssConfig, error) {
	if !strings.HasPrefix(uri, "ss://") {
		return nil, fmt.Errorf("invalid shadowsocks uri")
	}

	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	cfg := &ssConfig{
		Server: u.Host,
	}

	userInfo := u.User.String()
	if userInfo == "" {
		// Base64 encoded user info might be in the host part for some old URIs
		// but standard ones have it in the user part.
		// If user part is empty, check if host contains base64
		return nil, fmt.Errorf("missing user info in shadowsocks uri")
	}

	decodedUserInfo, err := base64.RawURLEncoding.DecodeString(userInfo)
	if err != nil {
		// Try standard base64 if raw url encoding fails
		decodedUserInfo, err = base64.StdEncoding.DecodeString(userInfo)
		if err != nil {
			// If both fail, assume it's NOT base64 encoded (plain text)
			decodedUserInfo = []byte(userInfo)
		}
	}

	parts := strings.SplitN(string(decodedUserInfo), ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid user info format")
	}

	cfg.Method = parts[0]
	cfg.Password = parts[1]

	return cfg, nil
}

func DialShadowsocks(confPath, host string, port int) (net.Conn, error) {
	// For now, confPath is the URI directly as specified in the plan
	cfg, err := parseSSUri(confPath)
	if err != nil {
		return nil, err
	}

	c, err := net.Dial("tcp", cfg.Server)
	if err != nil {
		return nil, err
	}

	ssConn, err := newSSConn(c, cfg.Method, cfg.Password)
	if err != nil {
		c.Close()
		return nil, err
	}

	// Send target address (ShadowSocks protocol)
	// Type: 3 (Domain Name)
	targetHost := host
	targetPort := uint16(port)

	var req []byte
	if ip := net.ParseIP(targetHost); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			req = append([]byte{1}, ip4...)
		} else {
			req = append([]byte{4}, ip.To16()...)
		}
	} else {
		req = append([]byte{3, byte(len(targetHost))}, []byte(targetHost)...)
	}
	req = append(req, byte(targetPort>>8), byte(targetPort))

	if _, err := ssConn.Write(req); err != nil {
		ssConn.Close()
		return nil, err
	}

	return ssConn, nil
}

func newSSConn(c net.Conn, method, password string) (net.Conn, error) {
	keyLen := 0
	switch strings.ToLower(method) {
	case "aes-128-gcm":
		keyLen = 16
	case "aes-256-gcm":
		keyLen = 32
	case "chacha20-ietf-poly1305":
		keyLen = 32
	default:
		return nil, fmt.Errorf("unsupported shadowsocks method: %s", method)
	}

	key := evpBytesToKey(password, keyLen)
	return &ssAEADConn{
		Conn:    c,
		method:  method,
		key:     key,
		readIV:  nil,
		writeIV: nil,
		reader:  nil,
		writer:  nil,
	}, nil
}

func evpBytesToKey(password string, keyLen int) []byte {
	var m []byte
	prev := []byte{}
	for len(m) < keyLen {
		h := sha1.New()
		h.Write(prev)
		h.Write([]byte(password))
		prev = h.Sum(nil)
		m = append(m, prev...)
	}
	return m[:keyLen]
}

type ssAEADConn struct {
	net.Conn
	method  string
	key     []byte
	readIV  []byte
	writeIV []byte
	reader  *aeadReader
	writer  *aeadWriter
}

func (c *ssAEADConn) Read(b []byte) (int, error) {
	if c.reader == nil {
		salt := make([]byte, len(c.key))
		if _, err := io.ReadFull(c.Conn, salt); err != nil {
			return 0, err
		}
		c.readIV = salt
		aead, err := c.createAEAD(salt)
		if err != nil {
			return 0, err
		}
		c.reader = &aeadReader{conn: c.Conn, aead: aead}
	}
	return c.reader.Read(b)
}

func (c *ssAEADConn) Write(b []byte) (int, error) {
	if c.writer == nil {
		salt := make([]byte, len(c.key))
		if _, err := io.ReadFull(rand.Reader, salt); err != nil {
			return 0, err
		}
		if _, err := c.Conn.Write(salt); err != nil {
			return 0, err
		}
		c.writeIV = salt
		aead, err := c.createAEAD(salt)
		if err != nil {
			return 0, err
		}
		c.writer = &aeadWriter{conn: c.Conn, aead: aead}
	}
	return c.writer.Write(b)
}

func (c *ssAEADConn) createAEAD(salt []byte) (cipher.AEAD, error) {
	subKey := make([]byte, len(c.key))
	info := []byte("ss-subkey")
	r := hkdf.New(sha1.New, c.key, salt, info)
	if _, err := io.ReadFull(r, subKey); err != nil {
		return nil, err
	}

	switch strings.ToLower(c.method) {
	case "aes-128-gcm", "aes-256-gcm":
		block, err := aes.NewCipher(subKey)
		if err != nil {
			return nil, err
		}
		return cipher.NewGCM(block)
	case "chacha20-ietf-poly1305":
		return chacha20poly1305.New(subKey)
	}
	return nil, fmt.Errorf("unsupported method")
}

type aeadReader struct {
	conn     net.Conn
	aead     cipher.AEAD
	nonce    []byte
	buf      []byte
	leftover []byte
}

func (r *aeadReader) Read(b []byte) (int, error) {
	if len(r.leftover) > 0 {
		n := copy(b, r.leftover)
		r.leftover = r.leftover[n:]
		return n, nil
	}

	if r.nonce == nil {
		r.nonce = make([]byte, r.aead.NonceSize())
	}

	// Read 2-byte length
	lenBuf := make([]byte, 2+r.aead.Overhead())
	if _, err := io.ReadFull(r.conn, lenBuf); err != nil {
		return 0, err
	}

	payloadLenBuf, err := r.aead.Open(nil, r.nonce, lenBuf, nil)
	if err != nil {
		return 0, err
	}
	incrementNonce(r.nonce)

	payloadLen := int(payloadLenBuf[0])<<8 | int(payloadLenBuf[1])
	payload := make([]byte, payloadLen+r.aead.Overhead())
	if _, err := io.ReadFull(r.conn, payload); err != nil {
		return 0, err
	}

	decrypted, err := r.aead.Open(nil, r.nonce, payload, nil)
	if err != nil {
		return 0, err
	}
	incrementNonce(r.nonce)

	n := copy(b, decrypted)
	if n < len(decrypted) {
		r.leftover = decrypted[n:]
	}
	return n, nil
}

type aeadWriter struct {
	conn  net.Conn
	aead  cipher.AEAD
	nonce []byte
}

func (r *aeadWriter) Write(b []byte) (int, error) {
	if r.nonce == nil {
		r.nonce = make([]byte, r.aead.NonceSize())
	}

	payloadLen := len(b)
	if payloadLen > 0x3FFF {
		payloadLen = 0x3FFF
	}

	// Write length
	lenBuf := []byte{byte(payloadLen >> 8), byte(payloadLen)}
	sealedLen := r.aead.Seal(nil, r.nonce, lenBuf, nil)
	incrementNonce(r.nonce)
	if _, err := r.conn.Write(sealedLen); err != nil {
		return 0, err
	}

	// Write payload
	sealedPayload := r.aead.Seal(nil, r.nonce, b[:payloadLen], nil)
	incrementNonce(r.nonce)
	if _, err := r.conn.Write(sealedPayload); err != nil {
		return 0, err
	}

	return payloadLen, nil
}

func incrementNonce(nonce []byte) {
	for i := range nonce {
		nonce[i]++
		if nonce[i] != 0 {
			break
		}
	}
}
