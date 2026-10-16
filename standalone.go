package main

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func startServer(handler http.Handler, httpAddr, tlsAddr, certFile, keyFile string) {
	ln, err := net.Listen("tcp", tlsAddr)
	if err != nil {
		log.Printf("TLS port %s is busy or unavailable (%v), running over HTTP on %s", tlsAddr, err, httpAddr)
		if err := http.ListenAndServe(httpAddr, handler); err != nil {
			log.Fatal(err)
		}
		return
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		ln.Close()
		log.Printf("Could not load certificate (%v), running over HTTP on %s", err, httpAddr)
		if err := http.ListenAndServe(httpAddr, handler); err != nil {
			log.Fatal(err)
		}
		return
	}
	tlsLn := tls.NewListener(ln, &tls.Config{Certificates: []tls.Certificate{cert}})
	server := &http.Server{Handler: handler}
	go func() {
		if err := http.ListenAndServe(httpAddr, handler); err != nil {
			log.Printf("HTTP listener crashed: %v", err)
		}
	}()
	log.Printf("TLS on %s, HTTP on %s", tlsAddr, httpAddr)
	if err := server.Serve(tlsLn); err != nil {
		log.Fatal(err)
	}
}

type noListingFS struct {
	fs http.FileSystem
}

func (n noListingFS) Open(name string) (http.File, error) {
	f, err := n.fs.Open(name)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if stat.IsDir() {
		f.Close()
		return nil, os.ErrNotExist
	}
	return f, nil
}

func staticHandler(prefix, dir, cacheControl, csp string) http.Handler {
	fileServer := http.FileServer(noListingFS{http.Dir(dir)})
	stripped := http.StripPrefix(prefix, fileServer)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Cache-Control", cacheControl)
		h.Set("X-Content-Type-Options", "nosniff")
		if csp != "" {
			h.Set("Content-Security-Policy", csp)
		}
		stripped.ServeHTTP(w, r)
	})
}

var dotfileRe = regexp.MustCompile(`/\.(ht|git)`)

func blockDotfiles(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if dotfileRe.MatchString(r.URL.Path) {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

var gzipTypes = map[string]bool{
	"text/plain":             true,
	"text/css":               true,
	"application/json":       true,
	"application/javascript": true,
	"text/xml":               true,
	"application/xml":        true,
	"text/javascript":        true,
}

const gzipMinLength = 2048

type bufferedWriter struct {
	http.ResponseWriter
	buf        *bytes.Buffer
	statusCode int
}

func (b *bufferedWriter) Write(p []byte) (int, error) {
	if b.buf == nil {
		b.buf = &bytes.Buffer{}
	}
	return b.buf.Write(p)
}

func (b *bufferedWriter) WriteHeader(code int) {
	b.statusCode = code
}

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		rec := &bufferedWriter{ResponseWriter: w, buf: &bytes.Buffer{}}
		next.ServeHTTP(rec, r)
		status := rec.statusCode
		if status == 0 {
			status = http.StatusOK
		}
		ct := w.Header().Get("Content-Type")
		base := strings.TrimSpace(strings.SplitN(ct, ";", 2)[0])
		if gzipTypes[base] && rec.buf.Len() >= gzipMinLength {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Del("Content-Length")
			w.WriteHeader(status)
			gz := gzip.NewWriter(w)
			gz.Write(rec.buf.Bytes())
			gz.Close()
			return
		}
		w.WriteHeader(status)
		w.Write(rec.buf.Bytes())
	})
}

func registerStaticRoutes(mux *http.ServeMux) {
	imgsDir := os.Getenv("IMGS")
	if imgsDir == "" {
		log.Fatal("IMGS is not set — without it /imgs/ serving will not come up")
	}
	versDir := os.Getenv("VERS_DIR")
	if versDir == "" {
		log.Fatal("VERS_DIR is not set — without it /cli/ serving will not come up")
	}
	const imgsCSP = "default-src 'none'; style-src 'unsafe-inline'; sandbox"
	mux.Handle("/imgs/", staticHandler("/imgs/", imgsDir, "public, no-transform, max-age=2592000", imgsCSP))
	mux.Handle("/cli/", staticHandler("/cli/", versDir, "public, no-transform, max-age=2592000", ""))
	// sw.js lies inside VERS_DIR (same place as the rest of the client code)
	mux.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, filepath.Join(versDir, "sw.js"))
	})
}
