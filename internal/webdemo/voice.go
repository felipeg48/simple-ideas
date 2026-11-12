package webdemo

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/felipeg48/simple-ideas/internal/home"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/tools"
)

// The narration: every Say: line is voiced before the take (its length sets
// the pace), normalised to 24 kHz mono 16-bit WAV and cached by the hash of
// engine, voice, style and the words said, so a re-run only voices the lines
// that changed. Local engines run as programs; cloud engines are plain HTTPS
// calls that send nothing but the Say: text.

// Voice is engine:name (name "" = the engine's default).
type Voice struct{ Engine, Name, Style string }

// Engines are the voice engines si knows.
var Engines = []string{"system", "kokoro", "elevenlabs", "openai"}

// defaultNames are each engine's voice when the script names none.
var defaultNames = map[string]string{"kokoro": "af_heart", "openai": "alloy", "elevenlabs": "21m00Tcm4TlvDq8ikWAM"}

// ParseVoice reads engine or engine:name.
func ParseVoice(s string) (Voice, error) {
	e, n, _ := strings.Cut(strings.TrimSpace(s), ":")
	e = strings.ToLower(strings.TrimSpace(e))
	for _, k := range Engines {
		if k == e {
			v := Voice{Engine: e, Name: strings.TrimSpace(n)}
			if v.Name == "" {
				v.Name = defaultNames[e]
			}
			return v, nil
		}
	}
	// a bare Kokoro voice name (af_heart, am_michael, bf_emma) is Kokoro's
	if n == "" && kokoroNameRE.MatchString(e) {
		return Voice{Engine: "kokoro", Name: e}, nil
	}
	return Voice{}, fmt.Errorf("voice %q: write engine:name, engine one of %s (kokoro:af_heart), or a Kokoro voice name (af_heart)", s, strings.Join(Engines, ", "))
}

var kokoroNameRE = regexp.MustCompile(`^[a-z]{2}_[a-z]+$`)

func (v Voice) String() string {
	if v.Name == "" {
		return v.Engine
	}
	return v.Engine + ":" + v.Name
}

// DefaultVoice is si's voice, Kokoro af_heart, when Kokoro is installed;
// else the computer's own.
func DefaultVoice() Voice {
	if tools.Find("kokoro") != "" {
		return Voice{Engine: "kokoro", Name: "af_heart"}
	}
	return Voice{Engine: "system"}
}

// Spoken is what the voice says for text: each pronounce: word (whole word,
// any case) replaced by how to say it.
func Spoken(text string, ps []Pronounce) string {
	for _, p := range ps {
		re := regexp.MustCompile(`(?i)(^|[^\pL\pN_])` + regexp.QuoteMeta(p.Word) + `($|[^\pL\pN_])`)
		for re.MatchString(text) { // overlapping neighbours ("si si") need a second pass
			text = re.ReplaceAllString(text, "${1}"+strings.ReplaceAll(p.Say, "$", "$$")+"${2}")
		}
	}
	return text
}

// cacheKey names a voiced line in the cache.
func cacheKey(v Voice, spoken string) string {
	h := sha256.Sum256([]byte(v.Engine + "\x00" + v.Name + "\x00" + v.Style + "\x00" + spoken))
	return hex.EncodeToString(h[:16])
}

// VoiceCache is the cache folder: ~/.si/cache/voice.
func VoiceCache() string { return home.Path("cache/voice", home.Cache, "voice") }

// Ready says why a voice cannot speak on this computer ("" when it can).
func (v Voice) Ready() string {
	switch v.Engine {
	case "kokoro":
		if tools.Find("kokoro") == "" {
			return tools.Need("kokoro", "voice: kokoro")
		}
	case "system":
		switch runtime.GOOS {
		case "darwin":
			if _, err := exec.LookPath("say"); err != nil {
				return "voice: system needs say (part of macOS)"
			}
		case "windows":
		default:
			if tools.Find("espeak-ng") == "" {
				return tools.Need("espeak-ng", "voice: system")
			}
		}
	case "elevenlabs":
		if k, _ := config.APIKey("elevenlabs"); k == "" {
			return "voice: elevenlabs needs an API key (elevenlabs.io, Profile › API keys): si voice login elevenlabs, or ELEVENLABS_API_KEY in the environment"
		}
	case "openai":
		if k, _ := config.APIKey("openai"); k == "" {
			return "voice: openai needs an API key (platform.openai.com › API keys): si voice login openai, or OPENAI_API_KEY in the environment"
		}
	}
	return ""
}

// KokoroVoices are the English voices in Kokoro's voices file (dir/voices-v1.0.bin,
// a zip of <name>.npy), or a built-in list when it cannot be read; si speaks
// American (a…) and British (b…) English.
func KokoroVoices(dir string) (names []string, fromFile bool) {
	if z, err := zip.OpenReader(filepath.Join(dir, "voices-v1.0.bin")); err == nil {
		defer z.Close()
		for _, f := range z.File {
			if n := strings.TrimSuffix(f.Name, ".npy"); n != f.Name && (n[0] == 'a' || n[0] == 'b') {
				names = append(names, n)
			}
		}
		if len(names) > 0 {
			sort.Strings(names)
			return names, true
		}
	}
	return strings.Fields(`af_alloy af_aoede af_bella af_heart af_jessica af_kore af_nicole af_nova af_river af_sarah af_sky
		am_adam am_echo am_eric am_fenrir am_liam am_michael am_onyx am_puck am_santa
		bf_alice bf_emma bf_isabella bf_lily bm_daniel bm_fable bm_george bm_lewis`), false
}

// Line is one voiced Say: line in the cache.
type Line struct {
	WAV     string
	Seconds float64
}

// Synth voices lines into the cache.
type Synth struct {
	FFmpeg string
	Cache  string
	HTTP   *http.Client
	Log    io.Writer
}

// cloud endpoints (tests point them at fakes)
var (
	elevenURL = "https://api.elevenlabs.io"
	openaiURL = "https://api.openai.com"
)

// Path is where text voiced by v is (or will be) in the cache.
func (sy *Synth) Path(v Voice, text string) string {
	return filepath.Join(sy.Cache, cacheKey(v, text)+".wav")
}

// Voice voices every text (already Spoken) with v and returns, per text, its
// WAV and length. new counts the lines that were not in the cache.
func (sy *Synth) Voice(v Voice, texts []string) (lines map[string]Line, new int, err error) {
	if err := os.MkdirAll(sy.Cache, 0o755); err != nil {
		return nil, 0, err
	}
	lines = map[string]Line{}
	var miss []string
	for _, t := range texts {
		if _, seen := lines[t]; seen || strings.TrimSpace(t) == "" {
			continue
		}
		p := sy.Path(v, t)
		if secs, err := wavSeconds(p); err == nil && secs > 0 {
			lines[t] = Line{p, secs}
			continue
		}
		lines[t] = Line{WAV: p}
		miss = append(miss, t)
	}
	if len(miss) == 0 {
		return lines, 0, nil
	}
	tmp, err := os.MkdirTemp("", "si-voice-")
	if err != nil {
		return nil, 0, err
	}
	defer os.RemoveAll(tmp)
	raw := make([]string, len(miss))
	for i := range miss {
		raw[i] = filepath.Join(tmp, fmt.Sprintf("raw%03d", i))
	}
	if err := sy.engine(v, miss, raw, tmp); err != nil {
		return nil, 0, err
	}
	for i, t := range miss {
		l := lines[t]
		if out, err := exec.Command(sy.FFmpeg, normArgs(raw[i], l.WAV)...).CombinedOutput(); err != nil {
			return nil, 0, fmt.Errorf("ffmpeg (voice): %v %s", err, strings.TrimSpace(string(out)))
		}
		if l.Seconds, err = wavSeconds(l.WAV); err != nil {
			return nil, 0, err
		}
		lines[t] = l
	}
	return lines, len(miss), nil
}

// normArgs turn any audio into the cache's WAV: 24 kHz mono 16-bit.
func normArgs(in, out string) []string {
	return []string{"-nostdin", "-y", "-hide_banner", "-loglevel", "error", "-i", in, "-ar", "24000", "-ac", "1", "-c:a", "pcm_s16le", out}
}

// engine writes one audio file per text to raw[i] (any format ffmpeg reads).
func (sy *Synth) engine(v Voice, texts, raw []string, tmp string) error {
	switch v.Engine {
	case "kokoro":
		return kokoro(v, texts, raw, tmp)
	case "system":
		for i, t := range texts {
			fmt.Fprintf(sy.log(), "  voice %d/%d\n", i+1, len(texts))
			txt := filepath.Join(tmp, fmt.Sprintf("t%03d.txt", i))
			if err := os.WriteFile(txt, []byte(t), 0o644); err != nil {
				return err
			}
			raw[i] += map[bool]string{true: ".aiff", false: ".wav"}[runtime.GOOS == "darwin"] // say picks its format by the name
			argv, env := systemArgs(runtime.GOOS, v.Name, txt, raw[i])
			if argv[0] == "espeak-ng" {
				if p := tools.Find("espeak-ng"); p != "" {
					argv[0] = p
				}
			}
			c := exec.Command(argv[0], argv[1:]...)
			c.Env = append(os.Environ(), env...)
			if out, err := c.CombinedOutput(); err != nil {
				return fmt.Errorf("voice: system (%s): %v %s", argv[0], err, strings.TrimSpace(string(out)))
			}
		}
		return nil
	case "elevenlabs", "openai":
		hc := sy.HTTP
		if hc == nil {
			hc = &http.Client{Timeout: 2 * time.Minute}
		}
		for i, t := range texts {
			fmt.Fprintf(sy.log(), "  voice %d/%d (%s)\n", i+1, len(texts), v.Engine)
			var req *http.Request
			var err error
			key, _ := config.APIKey(v.Engine)
			if v.Engine == "openai" {
				req, err = openaiRequest(v, t, key)
			} else {
				req, err = elevenRequest(v, t, key)
			}
			if err != nil {
				return err
			}
			body, err := fetch(hc, req)
			if err != nil {
				return fmt.Errorf("voice: %s: %v", v.Engine, err)
			}
			if v.Engine == "elevenlabs" { // raw PCM: give it a header
				body = pcmWAV(body, 24000)
			}
			if err := os.WriteFile(raw[i], body, 0o644); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("voice: no engine %q", v.Engine)
}

func (sy *Synth) log() io.Writer {
	if sy.Log == nil {
		return io.Discard
	}
	return sy.Log
}

// systemArgs is the command that voices the text in file txt into out with
// the computer's own voice, and the environment it needs.
func systemArgs(goos, name, txt, out string) ([]string, []string) {
	switch goos {
	case "darwin":
		a := []string{"say"}
		if name != "" {
			a = append(a, "-v", name)
		}
		return append(a, "-o", out, "-f", txt), nil
	case "windows":
		ps := `Add-Type -AssemblyName System.Speech;$s=New-Object System.Speech.Synthesis.SpeechSynthesizer;` +
			`if($env:SI_SAY_VOICE){$s.SelectVoice($env:SI_SAY_VOICE)};$s.SetOutputToWaveFile($env:SI_SAY_OUT);` +
			`$s.Speak([IO.File]::ReadAllText($env:SI_SAY_TEXT));$s.Dispose()`
		return []string{"powershell", "-NoProfile", "-NonInteractive", "-Command", ps},
			[]string{"SI_SAY_VOICE=" + name, "SI_SAY_OUT=" + out, "SI_SAY_TEXT=" + txt}
	}
	a := []string{"espeak-ng"}
	if name != "" {
		a = append(a, "-v", name)
	}
	return append(a, "-f", txt, "-w", out), nil
}

// kokoroPy voices a JSON list of {text, out} with kokoro-onnx in one process
// (the model loads once).
const kokoroPy = `import json, os, sys, wave
import numpy as np
from kokoro_onnx import Kokoro
model_dir, voice = sys.argv[1], sys.argv[2]
jobs = json.load(sys.stdin)
k = Kokoro(os.path.join(model_dir, "kokoro-v1.0.onnx"), os.path.join(model_dir, "voices-v1.0.bin"))
lang = "en-gb" if voice[:1] == "b" else "en-us"
for i, j in enumerate(jobs):
    samples, rate = k.create(j["text"], voice=voice, speed=1.0, lang=lang)
    pcm = (np.clip(samples, -1, 1) * 32767).astype("<i2").tobytes()
    with wave.open(j["out"], "wb") as w:
        w.setnchannels(1)
        w.setsampwidth(2)
        w.setframerate(rate)
        w.writeframes(pcm)
    print("  voice %d/%d (kokoro)" % (i + 1, len(jobs)), flush=True)
`

func kokoro(v Voice, texts, raw []string, tmp string) error {
	py := filepath.Join(tmp, "si-kokoro.py")
	if err := os.WriteFile(py, []byte(kokoroPy), 0o644); err != nil {
		return err
	}
	type job struct {
		Text string `json:"text"`
		Out  string `json:"out"`
	}
	var jobs []job
	for i, t := range texts {
		raw[i] += ".wav"
		jobs = append(jobs, job{t, raw[i]})
	}
	in, _ := json.Marshal(jobs)
	c := exec.Command(tools.KokoroPython(), py, tools.KokoroDir(), v.Name)
	c.Stdin = bytes.NewReader(in)
	if out, err := c.CombinedOutput(); err != nil {
		return fmt.Errorf("voice: kokoro: %v %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// elevenRequest asks ElevenLabs for raw 24 kHz PCM of text.
func elevenRequest(v Voice, text, key string) (*http.Request, error) {
	body, _ := json.Marshal(map[string]any{"text": text, "model_id": "eleven_multilingual_v2"})
	req, err := http.NewRequest("POST", elevenURL+"/v1/text-to-speech/"+v.Name+"?output_format=pcm_24000", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("xi-api-key", key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/pcm")
	return req, nil
}

// openaiRequest asks OpenAI for a WAV of text (gpt-4o-mini-tts, with the
// script's voice-style as its instructions).
func openaiRequest(v Voice, text, key string) (*http.Request, error) {
	m := map[string]any{"model": "gpt-4o-mini-tts", "voice": v.Name, "input": text, "response_format": "wav"}
	if v.Style != "" {
		m["instructions"] = v.Style
	}
	body, _ := json.Marshal(m)
	req, err := http.NewRequest("POST", openaiURL+"/v1/audio/speech", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func fetch(hc *http.Client, req *http.Request) ([]byte, error) {
	res, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 64<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode/100 != 2 {
		msg := strings.TrimSpace(string(b))
		if len(msg) > 300 {
			msg = msg[:300]
		}
		return nil, fmt.Errorf("%s: %s", res.Status, msg)
	}
	return b, nil
}

// pcmWAV wraps 16-bit mono PCM in a WAV header.
func pcmWAV(pcm []byte, rate int) []byte {
	var b bytes.Buffer
	w := func(v any) { binary.Write(&b, binary.LittleEndian, v) }
	b.WriteString("RIFF")
	w(uint32(36 + len(pcm)))
	b.WriteString("WAVEfmt ")
	w(uint32(16))
	w(uint16(1)) // PCM
	w(uint16(1)) // mono
	w(uint32(rate))
	w(uint32(rate * 2))
	w(uint16(2))
	w(uint16(16))
	b.WriteString("data")
	w(uint32(len(pcm)))
	b.Write(pcm)
	return b.Bytes()
}

// readWAV returns a 16-bit WAV's samples (mono expected) and rate.
func readWAV(path string) ([]int16, int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	if len(raw) < 12 || string(raw[:4]) != "RIFF" || string(raw[8:12]) != "WAVE" {
		return nil, 0, fmt.Errorf("%s: not a WAV file", path)
	}
	rate, bits := 0, 0
	for p := 12; p+8 <= len(raw); {
		id, n := string(raw[p:p+4]), int(binary.LittleEndian.Uint32(raw[p+4:p+8]))
		body := raw[p+8:]
		if n > len(body) {
			n = len(body)
		}
		switch id {
		case "fmt ":
			if n >= 16 {
				rate, bits = int(binary.LittleEndian.Uint32(body[4:8])), int(binary.LittleEndian.Uint16(body[14:16]))
			}
		case "data":
			if bits != 16 || rate == 0 {
				return nil, 0, fmt.Errorf("%s: not 16-bit PCM", path)
			}
			s := make([]int16, n/2)
			for i := range s {
				s[i] = int16(binary.LittleEndian.Uint16(body[2*i:]))
			}
			return s, rate, nil
		}
		p += 8 + n + n%2
	}
	return nil, 0, fmt.Errorf("%s: no audio", path)
}

func wavSeconds(path string) (float64, error) {
	s, rate, err := readWAV(path)
	if err != nil {
		return 0, err
	}
	return float64(len(s)) / float64(rate), nil
}

// placed is a voiced line at a time of the clip.
type placed struct {
	WAV string
	At  float64
}

// mixWAV writes total seconds of 24 kHz mono audio with each line at its time
// (lines that run past the end are cut there).
func mixWAV(out string, total float64, lines []placed) error {
	const rate = 24000
	buf := make([]int32, int(total*rate+.5))
	for _, l := range lines {
		s, r, err := readWAV(l.WAV)
		if err != nil {
			return err
		}
		if r != rate {
			return fmt.Errorf("%s: %d Hz, want %d", l.WAV, r, rate)
		}
		at := int(l.At*rate + .5)
		for i, v := range s {
			if at+i >= 0 && at+i < len(buf) {
				buf[at+i] += int32(v)
			}
		}
	}
	pcm := make([]byte, 2*len(buf))
	for i, v := range buf {
		binary.LittleEndian.PutUint16(pcm[2*i:], uint16(int16(max(-32768, min(32767, v)))))
	}
	return os.WriteFile(out, pcmWAV(pcm, rate), 0o644)
}
