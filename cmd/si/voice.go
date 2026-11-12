package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/tools"
	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

const voiceUsage = `Usage:
  si voice say "text" [--voice engine:name] [-o out.wav] [--play]
        say one line aloud, like macOS say, with the voice si demo run would use (--voice,
        else voice: in si.conf, else Kokoro af_heart when installed, else the computer's own).
        -o saves a WAV instead of playing it (--play: both). Lines are cached, so a repeated
        line plays at once. An engine that cannot speak here falls back to system, with a note.
  si voice list [kokoro|system|elevenlabs|openai]
        the voices each engine has
  si voice install kokoro [--dir D]
        set up Kokoro, the natural offline voice: a Python venv with kokoro-onnx and the two
        model files (about 350 MB, checked by SHA-256) in <user cache>/si/kokoro (or --dir D;
        then set SI_KOKORO_DIR=D). Needs Python 3.10 to 3.13. Run it again to repair.
  si voice uninstall kokoro [--dir D]
        remove what install put there
  si voice login elevenlabs|openai
        store the engine's API key in your own si.conf (` + "`si voice login`" + ` prints where), readable by
        you only; it asks without echo (or reads one line from a pipe). ELEVENLABS_API_KEY and
        OPENAI_API_KEY in the environment still win. Keys never go in a project's si.conf.
  si voice logout elevenlabs|openai
        remove the stored key
`

// voiceStdin is where login reads a key (tests swap it).
var voiceStdin io.Reader = os.Stdin

// lookPlayer finds an audio player on PATH (tests swap it, with runPlayer).
var lookPlayer = exec.LookPath

// runPlayer plays a WAV (tests swap it).
var runPlayer = func(argv []string) error {
	c := exec.Command(argv[0], argv[1:]...)
	c.Stderr = os.Stderr
	return c.Run()
}

// runVoice is `si voice …`, with its own flags.
func runVoice(argv []string, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("si voice", flag.ContinueOnError)
	fl.SetOutput(io.Discard)
	var voice, out, dir string
	var help, play bool
	fl.StringVar(&voice, "voice", "", "")
	fl.StringVar(&out, "output", "", "")
	fl.StringVar(&out, "o", "", "")
	fl.StringVar(&dir, "dir", "", "")
	fl.BoolVar(&play, "play", false, "")
	fl.BoolVar(&help, "help", false, "")
	fl.BoolVar(&help, "h", false, "")
	var args []string
	for {
		if err := fl.Parse(argv); err != nil {
			fmt.Fprintf(stderr, "si voice: %v\n\n%s", err, voiceUsage)
			return exitUsage
		}
		if argv = fl.Args(); len(argv) == 0 {
			break
		}
		args, argv = append(args, argv[0]), argv[1:]
	}
	if help {
		fmt.Fprint(stdout, voiceUsage)
		return exitOK
	}
	if len(args) == 0 {
		fmt.Fprintf(stderr, "si voice: say, list, install, uninstall, login or logout\n\n%s", voiceUsage)
		return exitUsage
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "say":
		if len(rest) == 0 {
			return usageVoice(stderr, `say needs the text: si voice say "Hello"`)
		}
		return voiceSay(strings.Join(rest, " "), voice, out, play, stdout, stderr)
	case "list":
		if len(rest) > 1 {
			return usageVoice(stderr, "list takes at most one engine")
		}
		return voiceList(rest, stdout, stderr)
	case "install", "uninstall":
		if len(rest) != 1 || rest[0] != "kokoro" {
			return usageVoice(stderr, cmd+" knows kokoro (the other engines need nothing installed)")
		}
		if dir == "" {
			dir = tools.KokoroDir()
		}
		if cmd == "uninstall" {
			if err := tools.UninstallKokoro(dir); err != nil {
				fmt.Fprintf(stderr, "si: %v\n", err)
				return exitProblems
			}
			fmt.Fprintf(stdout, "Kokoro removed from %s\n", dir)
			return exitOK
		}
		fmt.Fprintf(stderr, "si voice: installing Kokoro in %s\n", dir)
		if err := (tools.KokoroInstall{Dir: dir, Log: stderr}).Install(); err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
		fmt.Fprintf(stdout, "Kokoro is ready: si voice say \"Hello\" --voice kokoro:af_heart\n")
		// make it the default everywhere (si demo run, auto-play, si voice say),
		// unless the user's si.conf already names a voice
		if up := config.UserPath(); up != "" && filepath.Clean(dir) == filepath.Clean(tools.KokoroDir()) && config.Load(up).Voice == "" {
			if err := config.SetKey(up, "voice", "kokoro:af_heart"); err == nil {
				fmt.Fprintf(stdout, "Default voice: kokoro:af_heart (voice: in %s)\n", up)
			}
		}
		if filepath.Clean(dir) != filepath.Clean(tools.KokoroDir()) {
			fmt.Fprintf(stdout, "(it is in another folder: set SI_KOKORO_DIR=%s so si finds it)\n", dir)
		}
		return exitOK
	case "login", "logout":
		if len(rest) != 1 || config.KeyEnv[rest[0]] == "" {
			return usageVoice(stderr, cmd+" needs elevenlabs or openai")
		}
		if cmd == "logout" {
			return voiceLogout(rest[0], stdout, stderr)
		}
		return voiceLogin(rest[0], stdout, stderr)
	}
	return usageVoice(stderr, fmt.Sprintf("unknown command %q", cmd))
}

func usageVoice(stderr io.Writer, msg string) int {
	fmt.Fprintf(stderr, "si voice: %s\n\n%s", msg, voiceUsage)
	return exitUsage
}

// chooseVoice is the first voice set: the flag, then each of specs (front
// matter, si.conf), else webdemo.DefaultVoice.
func chooseVoice(specs ...string) (webdemo.Voice, error) {
	if s := firstNonEmpty(specs...); s != "" {
		return webdemo.ParseVoice(s)
	}
	return webdemo.DefaultVoice(), nil
}

// voiceSay voices text with the configured voice and plays it (and/or saves it).
func voiceSay(text, flagVoice, out string, play bool, stdout, stderr io.Writer) int {
	cfg := config.Load("", "si.conf")
	for _, w := range cfg.Warnings {
		fmt.Fprintf(stderr, "warning: %s\n", w)
	}
	v, err := chooseVoice(flagVoice, cfg.Voice)
	if err != nil {
		fmt.Fprintf(stderr, "si voice: %v\n", err)
		return exitUsage
	}
	v.Style = cfg.VoiceStyle
	if why := v.Ready(); why != "" {
		first, _, _ := strings.Cut(why, "\n")
		fmt.Fprintf(stderr, "si voice: %s; speaking with the system voice (si doctor)\n", strings.TrimSuffix(first, "."))
		v = webdemo.Voice{Engine: "system"}
		if why := v.Ready(); why != "" {
			fmt.Fprintf(stderr, "si: %s\n", why)
			return exitProblems
		}
	}
	sy := &webdemo.Synth{FFmpeg: tools.Find("ffmpeg"), Cache: webdemo.VoiceCache()}
	if _, err := os.Stat(sy.Path(v, text)); err != nil && sy.FFmpeg == "" {
		fmt.Fprintf(stderr, "si: %s\n", tools.Need("ffmpeg", "si voice say"))
		return exitProblems
	}
	lines, _, err := sy.Voice(v, []string{text})
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	wav := lines[text].WAV
	if out != "" {
		b, err := os.ReadFile(wav)
		if err == nil {
			err = os.WriteFile(out, b, 0o644)
		}
		if err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
		fmt.Fprintf(stdout, "%s (%s, %.1fs)\n", out, v, lines[text].Seconds)
	}
	if out == "" || play {
		argv := playerFor(runtime.GOOS, wav, lookPlayer)
		if argv == nil {
			fmt.Fprintf(stderr, "si voice: no audio player found (paplay, aplay or ffplay); the line is in %s\n", wav)
			return exitProblems
		}
		if err := runPlayer(argv); err != nil {
			fmt.Fprintf(stderr, "si voice: %s: %v\n", argv[0], err)
			return exitProblems
		}
	}
	return exitOK
}

// playerFor is the command that plays a WAV aloud and returns when it ends.
func playerFor(goos, wav string, look func(string) (string, error)) []string {
	switch goos {
	case "darwin":
		return []string{"afplay", wav}
	case "windows":
		return []string{"powershell", "-NoProfile", "-NonInteractive", "-Command",
			"(New-Object Media.SoundPlayer '" + strings.ReplaceAll(wav, "'", "''") + "').PlaySync()"}
	}
	for _, p := range [][]string{{"paplay", wav}, {"aplay", "-q", wav}, {"ffplay", "-nodisp", "-autoexit", "-loglevel", "quiet", wav}} {
		if _, err := look(p[0]); err == nil {
			return p
		}
	}
	return nil
}

// voiceList prints the voices of one engine, or of all.
func voiceList(args []string, stdout, stderr io.Writer) int {
	engines := webdemo.Engines
	if len(args) == 1 {
		v, err := webdemo.ParseVoice(args[0])
		if err != nil {
			fmt.Fprintf(stderr, "si voice: %v\n", err)
			return exitUsage
		}
		engines = []string{v.Engine}
	}
	for i, e := range engines {
		if i > 0 {
			fmt.Fprintln(stdout)
		}
		ready := "ready"
		if why := (webdemo.Voice{Engine: e}).Ready(); why != "" {
			ready, _, _ = strings.Cut(why, "\n")
		}
		fmt.Fprintf(stdout, "%s (%s)\n", e, ready)
		switch e {
		case "kokoro":
			names, fromFile := webdemo.KokoroVoices(tools.KokoroDir())
			fmt.Fprintf(stdout, "  %s\n", strings.Join(names, " "))
			if !fromFile {
				fmt.Fprintln(stdout, "  (si's list; si voice install kokoro to hear them)")
			}
			fmt.Fprintln(stdout, "  a… American, b… British; f female, m male. Default: kokoro:af_heart")
		case "system":
			if len(engines) > 1 {
				fmt.Fprintln(stdout, "  the computer's own voices: si voice list system")
				continue
			}
			argv := map[string][]string{
				"darwin": {"say", "-v", "?"},
				"windows": {"powershell", "-NoProfile", "-NonInteractive", "-Command",
					"Add-Type -AssemblyName System.Speech;(New-Object System.Speech.Synthesis.SpeechSynthesizer).GetInstalledVoices()|%{$_.VoiceInfo.Name}"},
			}[runtime.GOOS]
			if argv == nil {
				argv = []string{"espeak-ng", "--voices"}
			}
			c := exec.Command(argv[0], argv[1:]...)
			c.Stdout, c.Stderr = stdout, stderr
			if err := c.Run(); err != nil {
				fmt.Fprintf(stderr, "si voice: %s: %v\n", argv[0], err)
				return exitProblems
			}
			fmt.Fprintln(stdout, "  use one as system:<name> (system alone: the computer's default)")
		case "elevenlabs":
			fmt.Fprintln(stdout, "  any voice of your ElevenLabs account, by its id: elevenlabs:<voice id> (elevenlabs.io › Voices)")
			fmt.Fprintln(stdout, "  Default: elevenlabs:21m00Tcm4TlvDq8ikWAM (Rachel)")
		case "openai":
			fmt.Fprintln(stdout, "  alloy ash ballad coral echo fable nova onyx sage shimmer verse (or a newer name OpenAI adds)")
			fmt.Fprintln(stdout, "  Default: openai:alloy; voice.style: in si.conf or voice-style: in a script sets the tone")
		}
	}
	return exitOK
}

// voiceLogin stores an engine's API key in the user's si.conf, 0600.
func voiceLogin(engine string, stdout, stderr io.Writer) int {
	p := config.UserPath()
	if p == "" {
		fmt.Fprintln(stderr, "si: no user config folder on this computer")
		return exitProblems
	}
	key, err := readSecret(fmt.Sprintf("%s API key (it will not show): ", engine), stderr)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	if key == "" || strings.ContainsAny(key, " \t") {
		fmt.Fprintln(stderr, "si voice: no key given (one line, no spaces); nothing stored")
		return exitUsage
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	if f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600); err == nil {
		f.Close() // a new file starts private, before any key is in it
	}
	if err := os.Chmod(p, 0o600); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	if err := config.SetKey(p, "voice."+engine+".key", key); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	fmt.Fprintf(stdout, "logged in to %s: the key (%s) is in %s, readable by you only\n", engine, config.Mask(key), p)
	if env := config.KeyEnv[engine]; os.Getenv(env) != "" {
		fmt.Fprintf(stdout, "note: %s is set in this shell and still wins over the stored key\n", env)
	}
	return exitOK
}

func voiceLogout(engine string, stdout, stderr io.Writer) int {
	p := config.UserPath()
	if err := config.UnsetKey(p, "voice."+engine+".key"); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	fmt.Fprintf(stdout, "logged out of %s: no key for it in %s\n", engine, p)
	if env := config.KeyEnv[engine]; os.Getenv(env) != "" {
		fmt.Fprintf(stdout, "note: %s is still set in this shell\n", env)
	}
	return exitOK
}

// readSecret asks for one line: on a terminal without echo, from a pipe as is.
func readSecret(prompt string, stderr io.Writer) (string, error) {
	f, tty := voiceStdin.(*os.File)
	if tty {
		st, err := f.Stat()
		tty = err == nil && st.Mode()&os.ModeCharDevice != 0
	}
	if tty {
		fmt.Fprint(stderr, prompt)
		restore := noEcho(f)
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		go func() {
			if _, ok := <-sig; ok {
				restore()
				fmt.Fprintln(stderr)
				os.Exit(exitUsage)
			}
		}()
		defer func() {
			signal.Stop(sig)
			close(sig)
			restore()
			fmt.Fprintln(stderr)
		}()
	}
	line, err := bufio.NewReader(voiceStdin).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
