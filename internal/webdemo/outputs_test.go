package webdemo

import (
	"strings"
	"testing"
)

func TestWithAdd(t *testing.T) {
	for _, c := range []struct {
		lists []string
		want  With
		err   string
	}{
		{nil, With{}, ""},
		{[]string{""}, With{}, ""},
		{[]string{"captions, subtitles"}, With{Captions: true, Subtitles: true}, ""},
		{[]string{"captions", "subtitles"}, With{Captions: true, Subtitles: true}, ""},
		{[]string{"Clips,report", "readme"}, With{Clips: true, Report: true, Readme: true}, ""},
		{[]string{"all"}, With{true, true, true, true, true, true, true, true, true}, ""},
		{[]string{"silent,audio,transcript,chapters"}, With{Silent: true, Audio: true, Transcript: true, Chapters: true}, ""},
		{[]string{"captions,subtitels"}, With{}, `with "subtitels": use captions, silent, audio, subtitles, transcript, chapters, clips, report, readme, all`},
	} {
		var w With
		var err error
		for _, l := range c.lists {
			if err = w.Add(l); err != nil {
				break
			}
		}
		if c.err != "" {
			if err == nil || err.Error() != c.err {
				t.Errorf("%q: error %v, want %s", c.lists, err, c.err)
			}
			continue
		}
		if err != nil || w != c.want {
			t.Errorf("%q: %+v %v, want %+v", c.lists, w, err, c.want)
		}
	}
}

func TestWithFiles(t *testing.T) {
	voiced := []Planned{{"sign-in", true}, {"pay", false}}
	all := With{}
	all.Add("all")
	for _, c := range []struct {
		name string
		w    With
		fs   []Planned
		want string
	}{
		{"default", With{}, voiced, "shop.mp4"},
		{"nothing filmed", With{}, nil, ""},
		{"captions", With{Captions: true}, voiced, "shop.mp4 shop.captions.mp4"},
		{"no narration: no captions, audio, subtitles, transcript", With{Captions: true, Audio: true, Subtitles: true, Transcript: true, Silent: true}, []Planned{{"pay", false}}, "shop.mp4 shop.silent.mp4"},
		{"subtitles chapters", With{Subtitles: true, Chapters: true}, voiced, "shop.mp4 shop.srt shop.vtt chapters.txt"},
		{"clips", With{Clips: true}, voiced, "shop.mp4 sign-in.mp4 pay.mp4"},
		{"clips captions report", With{Clips: true, Captions: true, Report: true}, voiced,
			"shop.mp4 shop.captions.mp4 sign-in.mp4 sign-in.captions.mp4 pay.mp4 report.html report.json shots"},
		{"readme", With{Readme: true}, voiced, "shop.mp4 README.md README.html"},
		{"all", all, voiced, "shop.mp4 shop.captions.mp4 shop.silent.mp4 shop.m4a shop.srt shop.vtt shop.txt chapters.txt " +
			"sign-in.mp4 sign-in.captions.mp4 sign-in.silent.mp4 sign-in.m4a sign-in.srt sign-in.vtt pay.mp4 pay.silent.mp4 " +
			"report.html report.json shots README.md README.html"},
	} {
		if got := strings.Join(c.w.Files("shop", c.fs), " "); got != c.want {
			t.Errorf("%s:\n got %s\nwant %s", c.name, got, c.want)
		}
	}
}

func TestTranscript(t *testing.T) {
	s := Parse("/x/shop.md", "---\ntype: demo\napp: http://localhost:1\npronounce: si = ess eye\n---\n\n## Sign in\n\nSay: Open si and sign in.\n\n- go /\n\nSay: The dashboard comes up.\n\n- go /\n\n## Quiet\n\n- go /\n\n## Pay\n\nSay: Pay.\n\n- go /\n")
	if len(s.Problems) > 0 {
		t.Fatal(s.Problems)
	}
	want := "Sign in\n\nOpen si and sign in. The dashboard comes up.\n\nQuiet\n\nPay\n\nPay.\n"
	if got := transcript(s.Features); got != want {
		t.Errorf("transcript:\n%q\nwant\n%q", got, want)
	}
}

func TestWithFrontMatter(t *testing.T) {
	s := Parse("/x/shop.md", "---\ntype: demo\napp: http://localhost:1\nwith: captions, subtitels\n---\n\n## A\n\n- go /\n")
	if len(s.Problems) != 1 || s.Problems[0].Line != 4 || !strings.Contains(s.Problems[0].Msg, `with "subtitels"`) {
		t.Errorf("problems: %+v", s.Problems)
	}
	s = Parse("/x/shop.md", "---\ntype: demo\napp: http://localhost:1\nwith: captions, subtitles\n---\n\n## A\n\n- go /\n")
	if len(s.Problems) != 0 || s.With != "captions, subtitles" {
		t.Errorf("with: %q %+v", s.With, s.Problems)
	}
}
