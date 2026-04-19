package auth

import (
	"sync"
	"time"
)

// LoginLimiter begrenzt Login-Versuche pro IP-Adresse.
type LoginLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	max      int
	window   time.Duration
}

func NewLoginLimiter(max int, window time.Duration) *LoginLimiter {
	return &LoginLimiter{
		attempts: make(map[string][]time.Time),
		max:      max,
		window:   window,
	}
}

// Allow prüft ob ein Login-Versuch für die IP erlaubt ist.
func (l *LoginLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	// Abgelaufene Einträge entfernen
	valid := l.attempts[ip][:0]
	for _, t := range l.attempts[ip] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	l.attempts[ip] = valid

	if len(valid) >= l.max {
		return false
	}

	l.attempts[ip] = append(l.attempts[ip], now)
	return true
}

// Reset setzt den Zähler für eine IP zurück (nach erfolgreichem Login).
func (l *LoginLimiter) Reset(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, ip)
}
