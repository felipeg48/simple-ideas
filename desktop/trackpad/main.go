//go:build darwin

// si-trackpad turns a Mac's trackpad into a drawing tablet for a deck that
// si play serves: the pad maps to the whole display, a finger shows where
// the pen is, and a finger with ⌥ Option held draws. It reads finger
// positions (0..1 across the pad) from Apple's MultitouchSupport framework
// (private, but stable for years and used by many Mac apps), the Option key
// from CoreGraphics, freezes the cursor while Option is held, and posts "pad"
// messages to si play's relay on loopback. si play --trackpad starts it.
package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -F/System/Library/PrivateFrameworks -framework MultitouchSupport -framework CoreFoundation -framework CoreGraphics
#include <CoreFoundation/CoreFoundation.h>
#include <CoreGraphics/CoreGraphics.h>

typedef struct { float x, y; } mtPoint;
typedef struct { mtPoint pos, vel; } mtReadout;
typedef struct {
	int frame;
	double timestamp;
	int identifier, state, fingerID, handID;
	mtReadout normalized;
	float size;
	int zero1;
	float angle, majorAxis, minorAxis;
	mtReadout mm;
	int zero2[2];
	float unk2;
} Finger;
typedef void *MTDeviceRef;
typedef int (*MTContactCallbackFunction)(MTDeviceRef, Finger *, int, double, int);
MTDeviceRef MTDeviceCreateDefault(void);
void MTRegisterContactFrameCallback(MTDeviceRef, MTContactCallbackFunction);
void MTDeviceStart(MTDeviceRef, int);
void MTDeviceStop(MTDeviceRef);

extern void goFrame(int touching, float x, float y);

// the first finger in contact (states 3 make-touch, 4 touching), else the
// first finger near the pad, so the dot follows before the pen goes down
static int frameCB(MTDeviceRef dev, Finger *f, int n, double ts, int frame) {
	int pick = -1, touch = 0;
	for (int i = 0; i < n; i++) {
		if (f[i].state == 3 || f[i].state == 4) { pick = i; touch = 1; break; }
		if (pick < 0 && f[i].state > 0 && f[i].state < 7) pick = i;
	}
	if (pick < 0) goFrame(-1, 0, 0);
	else goFrame(touch, f[pick].normalized.pos.x, 1.0f - f[pick].normalized.pos.y);
	return 0;
}

static int startPad(void) {
	MTDeviceRef d = MTDeviceCreateDefault();
	if (!d) return 0;
	MTRegisterContactFrameCallback(d, frameCB);
	MTDeviceStart(d, 0);
	return 1;
}

static int optionDown(void) {
	return (CGEventSourceFlagsState(kCGEventSourceStateHIDSystemState) & kCGEventFlagMaskAlternate) != 0;
}

static void freezeCursor(int on) { CGAssociateMouseAndMouseCursorPosition(on ? false : true); }

// the active displays' bounds in global points, top-left origin, up to 16
static int displays(double *out) {
	CGDirectDisplayID ids[16];
	uint32_t n = 0;
	if (CGGetActiveDisplayList(16, ids, &n) != kCGErrorSuccess) return 0;
	for (uint32_t i = 0; i < n; i++) {
		CGRect r = CGDisplayBounds(ids[i]);
		out[i*4] = r.origin.x; out[i*4+1] = r.origin.y; out[i*4+2] = r.size.width; out[i*4+3] = r.size.height;
	}
	return (int)n;
}
*/
import "C"

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type frame struct {
	ok       bool // a finger is near or on the pad
	touching bool
	x, y     float64
}

var (
	mu     sync.Mutex
	latest frame
	fresh  bool
)

//export goFrame
func goFrame(touching C.int, x, y C.float) {
	mu.Lock()
	latest = frame{ok: touching >= 0, touching: touching == 1, x: float64(x), y: float64(y)}
	fresh = true
	mu.Unlock()
}

func screens() [][4]float64 {
	var buf [64]C.double
	n := int(C.displays(&buf[0]))
	out := make([][4]float64, n)
	for i := range out {
		out[i] = [4]float64{float64(buf[i*4]), float64(buf[i*4+1]), float64(buf[i*4+2]), float64(buf[i*4+3])}
	}
	return out
}

func main() {
	url := flag.String("url", "http://127.0.0.1:7070", "the si play address (loopback)")
	flag.Parse()
	if C.startPad() == 0 {
		fmt.Fprintln(os.Stderr, "si-trackpad: no trackpad found")
		os.Exit(1)
	}
	send := *url + "/_si/remote/send?c=trackpad"
	client := &http.Client{Timeout: 2 * time.Second}
	fmt.Fprintln(os.Stderr, "si-trackpad: the trackpad is the display; hold ⌥ Option and draw with one finger")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	tick := time.NewTicker(time.Second / 60)
	defer tick.Stop()
	frozen, seq := false, 0
	scr, scrAt := screens(), time.Now()
	for {
		select {
		case <-stop:
			C.freezeCursor(0)
			return
		case <-tick.C:
		}
		opt := C.optionDown() != 0
		optChanged := opt != frozen
		if optChanged { // the cursor stays put while Option is held, so the finger only draws
			C.freezeCursor(boolInt(opt))
			frozen = opt
		}
		mu.Lock()
		f, changed := latest, fresh
		fresh = false
		mu.Unlock()
		if !changed && !optChanged {
			continue
		}
		if time.Since(scrAt) > 2*time.Second { // displays come and go (a projector)
			scr, scrAt = screens(), time.Now()
		}
		seq++
		msg := map[string]any{"type": "pad", "mid": fmt.Sprintf("tp%d-%d", os.Getpid(), seq),
			"on": f.ok, "t": f.touching, "d": f.touching && opt, "x": round(f.x), "y": round(f.y), "s": scr}
		b, _ := json.Marshal(msg)
		if res, err := client.Post(send, "application/json", bytes.NewReader(b)); err == nil {
			res.Body.Close()
		}
	}
}

func boolInt(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

func round(v float64) float64 { return float64(int(v*10000+0.5)) / 10000 }
