package main

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// defaultPort is the fixed TCP port that board serve listens on and
// that board peers scans for. Chosen to be unlikely to collide with
// anything common on Android/Termux.
const defaultPort = 8848

// serveState bundles everything the HTTP handlers need.
type serveState struct {
	Priv      ed25519.PrivateKey
	Pub       ed25519.PublicKey
	PubB64    string
	Fp        string
	PublicDir string // ~/board-public/, served as static files
	Started   time.Time
}

// handleHello answers GET /board/hello with a fresh pointer so a peer
// can discover us and immediately know our pubkey and log state.
func (s *serveState) handleHello(w http.ResponseWriter, r *http.Request) {
	posts, err := readMyLog()
	if err != nil {
		http.Error(w, fmt.Sprintf("read log: %v", err), http.StatusInternalServerError)
		return
	}
	mirrors, _ := loadMirrors()
	p, err := buildPointer(s.Priv, posts, mirrors)
	if err != nil {
		// Log empty: return a minimal info response so a peer at least
		// sees us and knows our pubkey.
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"v":      1,
			"pubkey": s.PubB64,
			"fp":     s.Fp,
			"seq":    -1,
			"empty":  true,
			"ts":     time.Now().UTC().Format(time.RFC3339),
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(p)
}

// runServe starts the HTTP server and blocks. Returns only on error.
func runServe(port int) error {
	priv, err := loadOrCreateIdentity()
	if err != nil {
		return err
	}
	pub := priv.Public().(ed25519.PublicKey)
	pubB64 := pubkeyB64(pub)

	pubDir, err := publicDir()
	if err != nil {
		return err
	}

	s := &serveState{
		Priv:      priv,
		Pub:       pub,
		PubB64:    pubB64,
		Fp:        fingerprint(pub),
		PublicDir: pubDir,
		Started:   time.Now(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/board/hello", s.handleHello)
	// Serve the log directly from ~/.board/posts.log, always current.
	mux.HandleFunc("/posts.ndjson", func(w http.ResponseWriter, r *http.Request) {
		lp, err := myLogPath()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		http.ServeFile(w, r, lp)
	})

	// Serve everything else as static files from the public dir.
	// This covers pushed bundles and sent files.
	fileServer := http.FileServer(http.Dir(pubDir))
	mux.Handle("/", fileServer)

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("board serve\n")
	fmt.Printf("  fingerprint: %s\n", s.Fp)
	fmt.Printf("  pubkey:      %s\n", s.PubB64)
	fmt.Printf("  serving:     %s\n", s.PublicDir)
	fmt.Printf("  listening:   %s\n", addr)
	if ip, err := localIP(); err == nil {
		fmt.Printf("\nLocal mirror: http://%s:%d/posts.ndjson\n", ip, port)
		fmt.Printf("Discovery:    http://%s:%d/board/hello\n", ip, port)
	} else {
		fmt.Printf("\nDiscovery:    http://<your-lan-ip>:%d/board/hello\n", port)
	}
	fmt.Printf("Ctrl+C to stop.\n\n")

	if err := ensureSelfMirror(port); err != nil {
		fmt.Printf("warning: could not register self-mirror: %v\n", err)
	}
	if err := http.ListenAndServe(addr, logRequests(mux)); err != nil {
		return err
	}
	return nil
}

// logRequests wraps a handler with a small request log so the user can
// see who's connecting.
func logRequests(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		fmt.Printf("%s  %s %s  (%s)\n", time.Now().Format("15:04:05"), r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func ensureSelfMirror(port int) error {
	ip, err := localIP()
	if err != nil {
		return err
	}
	url := fmt.Sprintf("http://%s:%d/posts.ndjson", ip, port)
	mirrors, err := loadMirrors()
	if err != nil {
		return err
	}
	for _, m := range mirrors {
		if m == url {
			return nil
		}
	}
	mirrors = append(mirrors, url)
	return saveMirrors(mirrors)
}
