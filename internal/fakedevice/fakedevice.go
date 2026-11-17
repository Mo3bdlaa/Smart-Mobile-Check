// Package fakedevice emulates Apple's USB multiplexer service (usbmuxd /
// Apple Mobile Device Service) together with one connected iPhone. It speaks
// the same wire protocols as real hardware (usbmux, lockdown with TLS
// sessions, pairing, diagnostics_relay, installation_proxy, MCInstall, AFC and
// the crash-report services), so the real collector code can be exercised end
// to end in automated tests and demos without a physical device.
package fakedevice

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"

	ios "github.com/danielpaulus/go-ios/ios"
	"howett.net/plist"
)

const lockdownPort = 62078

// Device describes the emulated phone.
type Device struct {
	UDID        string
	Values      map[string]any            // lockdown "all values"
	Domains     map[string]map[string]any // lockdown domains
	Battery     map[string]any            // IORegistry AppleSmartBattery
	Diagnostics map[string]any            // diagnostics_relay "All"
	Apps        []map[string]any          // installation_proxy entries
	CloudConfig map[string]any
	Profiles    []Profile
	CrashFiles  map[string]string // crash-report area: path -> content
	FSModel     string
	FSTotal     uint64
	FSFree      uint64
	AFC2        bool // expose the jailbreak afc2 service
	// TrustAfter is how many Pair attempts show "Trust This Computer?"
	// before the user accepts. 0 means the device starts already trusted.
	TrustAfter int
}

// Profile is an installed configuration profile.
type Profile struct {
	ID, Name, Description string
	RemovalDisallowed     bool
}

// Server is a running emulator.
type Server struct {
	Dev *Device
	Log *log.Logger

	ln       net.Listener
	mu       sync.Mutex
	record   []byte // stored pair record plist (as saved by the host)
	parsed   ios.PairRecord
	attempts int
	devKey   *rsa.PrivateKey
}

var servicePorts = map[string]uint16{
	"com.apple.mobile.diagnostics_relay":  50001,
	"com.apple.mobile.installation_proxy": 50002,
	"com.apple.mobile.MCInstall":          50003,
	"com.apple.afc":                       50004,
	"com.apple.crashreportmover":          50005,
	"com.apple.crashreportcopymobile":     50006,
	"com.apple.afc2":                      50007,
}

var sslServices = map[string]bool{
	"com.apple.mobile.diagnostics_relay":  true,
	"com.apple.mobile.installation_proxy": true,
	"com.apple.mobile.MCInstall":          true,
}

// Start listens on addr ("127.0.0.1:0" for a random port).
func Start(addr string, dev *Device) (*Server, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	s := &Server{Dev: dev, ln: ln, devKey: key, Log: log.New(io.Discard, "", 0)}
	if dev.TrustAfter == 0 {
		if err := s.preTrust(); err != nil {
			ln.Close()
			return nil, err
		}
	}
	go s.accept()
	return s, nil
}

// Addr returns the listening address (host:port).
func (s *Server) Addr() string { return s.ln.Addr().String() }

// Close stops the emulator.
func (s *Server) Close() error { return s.ln.Close() }

// preTrust creates a pair record as if the host had paired earlier.
func (s *Server) preTrust() error {
	root, host, _, err := makeCerts()
	if err != nil {
		return err
	}
	rec := map[string]any{
		"HostID": "11111111-2222-3333-4444-555555555555", "SystemBUID": "AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE",
		"HostCertificate": host.cert, "HostPrivateKey": host.key, "RootCertificate": root.cert, "RootPrivateKey": root.key,
		"DeviceCertificate": host.cert, "EscrowBag": []byte("escrow"), "WiFiMACAddress": str(s.Dev.Values["WiFiAddress"]),
	}
	b, err := plist.Marshal(rec, plist.XMLFormat)
	if err != nil {
		return err
	}
	s.storeRecord(b)
	return nil
}

func (s *Server) storeRecord(b []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.record = b
	s.parsed = ios.PairRecordfromBytes(b)
}

func (s *Server) accept() {
	for {
		c, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handleMux(c)
	}
}

// ------------------------------------------------------------------ usbmux

func (s *Server) handleMux(c net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Printf("mux panic: %v", r)
		}
	}()
	for {
		var h ios.UsbMuxHeader
		if err := binary.Read(c, binary.LittleEndian, &h); err != nil {
			c.Close()
			return
		}
		body := make([]byte, h.Length-16)
		if _, err := io.ReadFull(c, body); err != nil {
			c.Close()
			return
		}
		msg, _ := ios.ParsePlist(body)
		mt := str(msg["MessageType"])
		s.Log.Printf("usbmux %s", mt)
		reply := func(v any) { s.muxSend(c, h.Tag, v) }
		switch mt {
		case "ListDevices":
			reply(map[string]any{"DeviceList": []any{map[string]any{
				"DeviceID": 1, "MessageType": "Attached",
				"Properties": map[string]any{"ConnectionType": "USB", "DeviceID": 1, "SerialNumber": s.Dev.UDID, "ProductID": 4776, "LocationID": 0, "ConnectionSpeed": 480000000},
			}}})
		case "ReadBUID":
			reply(map[string]any{"BUID": "AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE"})
		case "ReadPairRecord":
			s.mu.Lock()
			rec := s.record
			s.mu.Unlock()
			if rec == nil || str(msg["PairRecordID"]) != s.Dev.UDID {
				reply(map[string]any{"MessageType": "Result", "Number": 2})
			} else {
				reply(map[string]any{"PairRecordData": rec})
			}
		case "SavePairRecord":
			if b, ok := msg["PairRecordData"].([]byte); ok {
				s.storeRecord(b)
			}
			reply(map[string]any{"MessageType": "Result", "Number": 0})
		case "Connect":
			pn, _ := toU(msg["PortNumber"])
			port := ios.Ntohs(uint16(pn))
			if port != lockdownPort && !knownPort(port) {
				reply(map[string]any{"MessageType": "Result", "Number": 3})
				continue
			}
			reply(map[string]any{"MessageType": "Result", "Number": 0})
			s.route(c, port)
			return
		default:
			reply(map[string]any{"MessageType": "Result", "Number": 0})
		}
	}
}

func knownPort(p uint16) bool {
	for _, v := range servicePorts {
		if v == p {
			return true
		}
	}
	return false
}

func (s *Server) muxSend(c net.Conn, tag uint32, v any) {
	b, err := plist.Marshal(v, plist.XMLFormat)
	if err != nil {
		return
	}
	h := ios.UsbMuxHeader{Length: uint32(16 + len(b)), Version: 1, Request: 8, Tag: tag}
	_ = binary.Write(c, binary.LittleEndian, h)
	_, _ = c.Write(b)
}

func (s *Server) route(c net.Conn, port uint16) {
	dc := ios.NewDeviceConnectionWithConn(c)
	if port == lockdownPort {
		s.lockdown(dc)
		return
	}
	var name string
	for k, v := range servicePorts {
		if v == port {
			name = k
		}
	}
	if sslServices[name] {
		s.mu.Lock()
		rec := s.parsed
		s.mu.Unlock()
		if err := dc.EnableSessionSslServerMode(rec); err != nil {
			s.Log.Printf("service TLS failed: %v", err)
			dc.Close()
			return
		}
	}
	defer dc.Close()
	switch name {
	case "com.apple.mobile.diagnostics_relay":
		s.diagnostics(dc)
	case "com.apple.mobile.installation_proxy":
		s.installProxy(dc)
	case "com.apple.mobile.MCInstall":
		s.mcinstall(dc)
	case "com.apple.crashreportmover":
		_, _ = dc.Writer().Write([]byte("ping"))
	case "com.apple.crashreportcopymobile":
		s.afc(dc, s.Dev.CrashFiles)
	case "com.apple.afc", "com.apple.afc2":
		s.afc(dc, map[string]string{"DCIM/100APPLE/IMG_0001.HEIC": "img"})
	}
}

// ---------------------------------------------------------------- lockdown

func (s *Server) lockdown(dc *ios.DeviceConnection) {
	defer dc.Close()
	codec := ios.NewPlistCodec()
	session := false
	send := func(v any) {
		b, err := codec.Encode(v)
		if err == nil {
			_ = dc.Send(b)
		}
	}
	for {
		raw, err := codec.Decode(dc.Reader())
		if err != nil {
			return
		}
		req, _ := ios.ParsePlist(raw)
		r := str(req["Request"])
		s.Log.Printf("lockdown %s %v/%v session=%v", r, req["Domain"], req["Key"], session)
		switch r {
		case "QueryType":
			send(map[string]any{"Request": r, "Type": "com.apple.mobile.lockdown"})
		case "GetValue":
			send(s.getValue(req, session))
		case "Pair":
			s.mu.Lock()
			s.attempts++
			ok := s.attempts > s.Dev.TrustAfter
			s.mu.Unlock()
			if !ok {
				send(map[string]any{"Request": r, "Error": "PairingDialogResponsePending"})
			} else {
				send(map[string]any{"Request": r, "EscrowBag": []byte("escrow-bag")})
			}
		case "StartSession":
			s.mu.Lock()
			rec := s.parsed
			have := s.record != nil
			s.mu.Unlock()
			if !have || str(req["HostID"]) != rec.HostID {
				send(map[string]any{"Request": r, "Error": "InvalidHostID"})
				continue
			}
			send(map[string]any{"Request": r, "EnableSessionSSL": true, "SessionID": "SESSION-1"})
			if err := dc.EnableSessionSslServerMode(rec); err != nil {
				s.Log.Printf("lockdown TLS failed: %v", err)
				return
			}
			session = true
		case "StopSession":
			send(map[string]any{"Request": r})
		case "StartService":
			name := str(req["Service"])
			port, ok := servicePorts[name]
			if !session || !ok || (name == "com.apple.afc2" && !s.Dev.AFC2) {
				send(map[string]any{"Request": r, "Service": name, "Error": "InvalidService"})
				continue
			}
			send(map[string]any{"Request": r, "Service": name, "Port": port, "EnableServiceSSL": sslServices[name]})
		case "Goodbye":
			return
		default:
			send(map[string]any{"Request": r, "Error": "UnknownRequest"})
		}
	}
}

var preSessionKeys = map[string]bool{
	"ProductType": true, "ProductVersion": true, "BuildVersion": true, "DeviceClass": true,
	"UniqueDeviceID": true, "DevicePublicKey": true, "WiFiAddress": true, "ProtocolVersion": true,
}

func (s *Server) getValue(req map[string]any, session bool) map[string]any {
	domain, key := str(req["Domain"]), str(req["Key"])
	resp := map[string]any{"Request": "GetValue"}
	if domain != "" {
		resp["Domain"] = domain
	}
	if key != "" {
		resp["Key"] = key
	}
	all := map[string]any{}
	for k, v := range s.Dev.Values {
		all[k] = v
	}
	all["DevicePublicKey"] = pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&s.devKey.PublicKey)})
	if !session {
		if domain != "" || key == "" || !preSessionKeys[key] {
			resp["Error"] = "MissingValue"
			return resp
		}
	}
	var src map[string]any
	if domain == "" {
		src = all
	} else {
		src = s.Dev.Domains[domain]
	}
	if src == nil {
		resp["Error"] = "MissingValue"
		return resp
	}
	if key == "" {
		resp["Value"] = src
		return resp
	}
	v, ok := src[key]
	if !ok {
		resp["Error"] = "MissingValue"
		return resp
	}
	resp["Value"] = v
	return resp
}

// ---------------------------------------------------------------- services

func plistLoop(dc *ios.DeviceConnection, handle func(map[string]any) (any, bool)) {
	codec := ios.NewPlistCodec()
	for {
		raw, err := codec.Decode(dc.Reader())
		if err != nil {
			return
		}
		req, _ := ios.ParsePlist(raw)
		resp, more := handle(req)
		if resp != nil {
			if b, err := codec.Encode(resp); err == nil {
				_ = dc.Send(b)
			}
		}
		if !more {
			return
		}
	}
}

func (s *Server) diagnostics(dc *ios.DeviceConnection) {
	plistLoop(dc, func(req map[string]any) (any, bool) {
		switch str(req["Request"]) {
		case "IORegistry":
			name, class := str(req["EntryName"]), str(req["EntryClass"])
			reg := map[string]any{}
			if name == "AppleSmartBattery" || class == "IOPMPowerSource" {
				reg = s.Dev.Battery
			}
			return map[string]any{"Status": "Success", "Diagnostics": map[string]any{"IORegistry": reg}}, true
		case "All":
			return map[string]any{"Status": "Success", "Diagnostics": s.Dev.Diagnostics}, true
		case "MobileGestalt":
			// iOS 17.4+ behaviour.
			return map[string]any{"Status": "MobileGestaltDeprecated"}, true
		case "Goodbye":
			return map[string]any{"Status": "Success"}, false
		}
		return map[string]any{"Status": "UnknownRequest"}, true
	})
}

func (s *Server) installProxy(dc *ios.DeviceConnection) {
	codec := ios.NewPlistCodec()
	plistLoop(dc, func(req map[string]any) (any, bool) {
		if str(req["Command"]) != "Browse" {
			return map[string]any{"Error": "UnknownCommand"}, true
		}
		list := make([]any, 0, len(s.Dev.Apps))
		for _, a := range s.Dev.Apps {
			list = append(list, a)
		}
		// Send in two chunks to exercise the client's chunk assembly.
		half := len(list) / 2
		for i, part := range [][]any{list[:half], list[half:]} {
			idx := 0
			if i == 1 {
				idx = half
			}
			b, _ := codec.Encode(map[string]any{"Status": "BrowsingApplications", "CurrentIndex": idx, "CurrentAmount": len(part), "CurrentList": part})
			_ = dc.Send(b)
		}
		return map[string]any{"Status": "Complete"}, true
	})
}

func (s *Server) mcinstall(dc *ios.DeviceConnection) {
	plistLoop(dc, func(req map[string]any) (any, bool) {
		switch str(req["RequestType"]) {
		case "GetCloudConfiguration":
			cc := s.Dev.CloudConfig
			if cc == nil {
				cc = map[string]any{}
			}
			return map[string]any{"Status": "Acknowledged", "CloudConfiguration": cc}, true
		case "GetProfileList":
			ids := []any{}
			man := map[string]any{}
			meta := map[string]any{}
			for i, p := range s.Dev.Profiles {
				ids = append(ids, p.ID)
				man[p.ID] = map[string]any{"IsActive": true, "Description": p.Description}
				meta[p.ID] = map[string]any{"PayloadDisplayName": p.Name, "PayloadDescription": p.Description,
					"PayloadRemovalDisallowed": p.RemovalDisallowed, "PayloadUUID": fmt.Sprintf("UUID-%d", i), "PayloadVersion": uint64(1)}
			}
			return map[string]any{"Status": "Acknowledged", "OrderedIdentifiers": ids, "ProfileManifest": man, "ProfileMetadata": meta}, true
		}
		return map[string]any{"Status": "Error"}, true
	})
}

// ---------------------------------------------------------------- AFC

const (
	afcMagic      uint64 = 0x4141504c36414643
	opStatus             = 0x01
	opData               = 0x02
	opReadDir            = 0x03
	opFileInfo           = 0x0A
	opDeviceInfo         = 0x0B
	opFileOpen           = 0x0D
	opFileOpenRes        = 0x0E
	opFileRead           = 0x0F
	opFileClose          = 0x14
	afcNotFound          = 8
)

type afcHeader struct {
	Magic, EntireLen, ThisLen, PacketNum, Operation uint64
}

func (s *Server) afc(dc *ios.DeviceConnection, files map[string]string) {
	rw := dc.Conn()
	norm := func(p string) string {
		p = path.Clean("/" + strings.TrimRight(p, "\x00"))
		return strings.TrimPrefix(p, "/")
	}
	isDir := func(p string) bool {
		if p == "" {
			return true
		}
		for f := range files {
			if strings.HasPrefix(f, p+"/") {
				return true
			}
		}
		return false
	}
	type handle struct {
		data []byte
		off  int
	}
	handles := map[uint64]*handle{}
	next := uint64(1)
	write := func(num, op uint64, hp, payload []byte) {
		h := afcHeader{afcMagic, uint64(40 + len(hp) + len(payload)), uint64(40 + len(hp)), num, op}
		var buf bytes.Buffer
		_ = binary.Write(&buf, binary.LittleEndian, h)
		buf.Write(hp)
		buf.Write(payload)
		_, _ = rw.Write(buf.Bytes())
	}
	status := func(num, code uint64) {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, code)
		write(num, opStatus, b, nil)
	}
	for {
		var h afcHeader
		if err := binary.Read(rw, binary.LittleEndian, &h); err != nil {
			return
		}
		hp := make([]byte, h.ThisLen-40)
		pl := make([]byte, h.EntireLen-h.ThisLen)
		if _, err := io.ReadFull(rw, hp); err != nil {
			return
		}
		if _, err := io.ReadFull(rw, pl); err != nil {
			return
		}
		switch h.Operation {
		case opDeviceInfo:
			kv := []string{"Model", s.Dev.FSModel, "FSTotalBytes", strconv.FormatUint(s.Dev.FSTotal, 10),
				"FSFreeBytes", strconv.FormatUint(s.Dev.FSFree, 10), "FSBlockSize", "4096"}
			write(h.PacketNum, opData, nil, []byte(strings.Join(kv, "\x00")+"\x00"))
		case opReadDir:
			p := norm(string(hp))
			if !isDir(p) {
				status(h.PacketNum, afcNotFound)
				continue
			}
			seen := map[string]bool{}
			names := []string{".", ".."}
			for f := range files {
				rel := f
				if p != "" {
					if !strings.HasPrefix(f, p+"/") {
						continue
					}
					rel = strings.TrimPrefix(f, p+"/")
				}
				first := strings.SplitN(rel, "/", 2)[0]
				if !seen[first] {
					seen[first] = true
					names = append(names, first)
				}
			}
			sort.Strings(names[2:])
			write(h.PacketNum, opData, nil, []byte(strings.Join(names, "\x00")+"\x00"))
		case opFileInfo:
			p := norm(string(hp))
			if c, ok := files[p]; ok {
				write(h.PacketNum, opData, nil, []byte("st_size\x00"+strconv.Itoa(len(c))+"\x00st_ifmt\x00S_IFREG\x00st_mode\x00100644\x00"))
			} else if isDir(p) {
				write(h.PacketNum, opData, nil, []byte("st_size\x00"+"96\x00st_ifmt\x00S_IFDIR\x00st_mode\x0040755\x00"))
			} else {
				status(h.PacketNum, afcNotFound)
			}
		case opFileOpen:
			p := norm(string(hp[8:]))
			c, ok := files[p]
			if !ok {
				status(h.PacketNum, afcNotFound)
				continue
			}
			fd := next
			next++
			handles[fd] = &handle{data: []byte(c)}
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, fd)
			write(h.PacketNum, opFileOpenRes, b, nil)
		case opFileRead:
			fd := binary.LittleEndian.Uint64(hp)
			n := int(binary.LittleEndian.Uint64(hp[8:]))
			hd := handles[fd]
			if hd == nil {
				status(h.PacketNum, afcNotFound)
				continue
			}
			end := hd.off + n
			if end > len(hd.data) {
				end = len(hd.data)
			}
			chunk := hd.data[hd.off:end]
			hd.off = end
			write(h.PacketNum, opData, nil, chunk)
		case opFileClose:
			delete(handles, binary.LittleEndian.Uint64(hp))
			status(h.PacketNum, 0)
		default:
			status(h.PacketNum, 0)
		}
	}
}

// ---------------------------------------------------------------- helpers

type keyCert struct{ cert, key []byte }

func makeCerts() (root, host keyCert, dev keyCert, err error) {
	gen := func(cn string) (keyCert, error) {
		k, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return keyCert{}, err
		}
		tpl := x509.Certificate{SerialNumber: big.NewInt(1), BasicConstraintsValid: true, IsCA: cn == "root"}
		tpl.NotBefore = tpl.NotBefore.AddDate(2000, 0, 0)
		tpl.NotAfter = tpl.NotBefore.AddDate(100, 0, 0)
		der, err := x509.CreateCertificate(rand.Reader, &tpl, &tpl, &k.PublicKey, k)
		if err != nil {
			return keyCert{}, err
		}
		return keyCert{
			cert: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
			key:  pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}),
		}, nil
	}
	if root, err = gen("root"); err != nil {
		return
	}
	host, err = gen("host")
	dev = host
	return
}

func str(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func toU(v any) (uint64, bool) {
	switch t := v.(type) {
	case uint64:
		return t, true
	case int64:
		return uint64(t), true
	case int:
		return uint64(t), true
	}
	return 0, false
}
