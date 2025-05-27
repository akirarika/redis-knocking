package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type cacheItem struct {
	allowed bool
	expire  time.Time
}

var (
	redisClient *redis.Client
	ipCache     = struct {
		sync.RWMutex
		items map[string]cacheItem
	}{
		items: make(map[string]cacheItem),
	}
)

func runScript(script string) {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell.exe", "-Command", script)
	} else {
		cmd = exec.Command("bash", "-c", script)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}

	go func() {
		err := cmd.Wait()
		exitCode := 0
		if err != nil {
			log.Printf("Command exited with error: %v", err)
			if exiterr, ok := err.(*exec.ExitError); ok {
				exitCode = exiterr.ExitCode()
			} else {
				exitCode = 1
			}
		} else {
			log.Printf("Command exited successfully")
		}
		os.Exit(exitCode)
	}()
}

func getClientIP(r *http.Request, ipHeader string) string {
	if ipHeader != "" {
		if headerValue := r.Header.Get(ipHeader); headerValue != "" {
			ips := strings.Split(headerValue, ",")
			if len(ips) > 0 {
				return strings.TrimSpace(ips[0])
			}
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func isInternalIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	if ip.IsLoopback() {
		return true
	}

	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168)
	}

	if ip6 := ip.To16(); ip6 != nil {
		return len(ip6) >= 2 && (ip6[0]&0xFE) == 0xFC
	}

	return false
}

func respondDenied(w http.ResponseWriter, r *http.Request, redirectURL string) {
	if redirectURL != "" {
		http.Redirect(w, r, redirectURL, http.StatusFound)
	} else {
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		
		conn, _, err := hj.Hijack()
		if err != nil {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		
		conn.Close()
	}
}

func main() {
	script := flag.String("script", "echo '🚨 No command was specified to run, please add the -script parameter.'", "Execute some commands to start the server")
	listenAddr := flag.String("listen", ":8081", "Listen address and port")
	targetAddr := flag.String("target", "http://localhost:8080", "Target server address")
	redisURL := flag.String("redis", "", "Redis connection URL (e.g. redis://:password@host:port/db)")
	redisSetKey := flag.String("set", "ip-allowed", "Redis set key containing allowed IPs")
	ipHeader := flag.String("ip-header", "", "HTTP header to get client IP (e.g. X-Real-IP)")
	redirect := flag.String("redirect", "", "Redirect URL for denied requests (sends 302 if set)")
	detail := flag.String("detail", "disabled", "Display more information in the console for debugging")
	flag.Parse()

	if *redisURL == "" || *redisSetKey == "" {
		log.Fatal("Redis URL and set key are required")
	}

	opt, err := redis.ParseURL(*redisURL)
	if err != nil {
		log.Fatalf("Invalid Redis URL: %v", err)
	}

	redisClient = redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	targetUrl, err := url.Parse(*targetAddr)
	if err != nil {
		log.Fatalf("Invalid target address: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetUrl)
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = targetUrl.Scheme
		req.URL.Host = targetUrl.Host
		req.Host = targetUrl.Host

		if req.Header.Get("Connection") == "Upgrade" {
			req.Header.Set("Connection", "upgrade")
		}
	}

	runScript(*script)

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			now := time.Now()
			ipCache.Lock()
			for ip, item := range ipCache.items {
				if now.After(item.expire) {
					delete(ipCache.items, ip)
				}
			}
			ipCache.Unlock()
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r, *ipHeader)
		if isInternalIP(clientIP) {
			if *detail == "enabled" {
				log.Printf("Internal IP %s automatically allowed", clientIP)
			}
			proxy.ServeHTTP(w, r)
			return
		}

		ipCache.RLock()
		item, found := ipCache.items[clientIP]
		ipCache.RUnlock()

		if found && time.Now().Before(item.expire) {
			if item.allowed {
				if *detail == "enabled" {
					log.Printf("IP %s allowed (cached)", clientIP)
				}
				proxy.ServeHTTP(w, r)
			} else {
				if *detail == "enabled" {
					log.Printf("IP %s denied (cached)", clientIP)
				}
				respondDenied(w, r, *redirect)
			}
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		isMember, err := redisClient.SIsMember(ctx, *redisSetKey, clientIP).Result()
		if err != nil {
			log.Printf("Redis error: %v", err)
			respondDenied(w, r, *redirect)
			os.Exit(1)
		}

		allowed := isMember
		ipCache.Lock()
		ipCache.items[clientIP] = cacheItem{
			allowed: allowed,
			expire:  time.Now().Add(10 * time.Second),
		}
		ipCache.Unlock()

		if allowed {
			if *detail == "enabled" {
				log.Printf("IP %s allowed", clientIP)
			}
			proxy.ServeHTTP(w, r)
		} else {
			if *detail == "enabled" {
				log.Printf("IP %s denied", clientIP)
			}
			respondDenied(w, r, *redirect)
		}
	})

	log.Printf("Port forwarding service is running: %s -> %s", *listenAddr, *targetAddr)
	log.Fatal(http.ListenAndServe(*listenAddr, nil))
}