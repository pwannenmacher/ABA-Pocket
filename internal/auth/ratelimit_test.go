package auth

import (
	"testing"
	"time"
)

func TestLoginLimiterAllow(t *testing.T) {
	l := NewLoginLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		if !l.Allow("1.2.3.4") {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
	}
	if l.Allow("1.2.3.4") {
		t.Error("4th attempt should be blocked")
	}
}

func TestLoginLimiterDifferentIPs(t *testing.T) {
	l := NewLoginLimiter(1, time.Minute)
	if !l.Allow("1.1.1.1") {
		t.Error("first IP should be allowed")
	}
	if !l.Allow("2.2.2.2") {
		t.Error("second IP should be allowed independently")
	}
	if l.Allow("1.1.1.1") {
		t.Error("first IP should now be blocked")
	}
}

func TestLoginLimiterReset(t *testing.T) {
	l := NewLoginLimiter(1, time.Minute)
	l.Allow("1.2.3.4")
	if l.Allow("1.2.3.4") {
		t.Error("should be blocked before reset")
	}
	l.Reset("1.2.3.4")
	if !l.Allow("1.2.3.4") {
		t.Error("should be allowed after reset")
	}
}

func TestLoginLimiterWindowExpiry(t *testing.T) {
	l := NewLoginLimiter(1, 10*time.Millisecond)
	l.Allow("1.2.3.4")
	if l.Allow("1.2.3.4") {
		t.Error("should be blocked immediately")
	}
	time.Sleep(20 * time.Millisecond)
	if !l.Allow("1.2.3.4") {
		t.Error("should be allowed after window expires")
	}
}
