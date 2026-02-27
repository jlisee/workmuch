package app

import "testing"

func TestParseOptionsDefaults(t *testing.T) {
	opts, showHelp, err := ParseOptions(nil)
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}
	if showHelp {
		t.Fatalf("showHelp should be false by default")
	}
	if opts.Rate != 1.0 {
		t.Fatalf("unexpected default rate: %f", opts.Rate)
	}
	if opts.StartDelay != 0.0 {
		t.Fatalf("unexpected default delay: %f", opts.StartDelay)
	}
}

func TestParseOptionsShortFlags(t *testing.T) {
	opts, showHelp, err := ParseOptions([]string{"-r", "2.5", "-d", "1.25", "--qa-console"})
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}
	if showHelp {
		t.Fatalf("showHelp should be false")
	}
	if opts.Rate != 2.5 {
		t.Fatalf("unexpected rate: %f", opts.Rate)
	}
	if opts.StartDelay != 1.25 {
		t.Fatalf("unexpected delay: %f", opts.StartDelay)
	}
	if !opts.QAConsole {
		t.Fatalf("expected QA console true")
	}
}

func TestParseOptionsLongFlags(t *testing.T) {
	opts, _, err := ParseOptions([]string{"--rate=3", "--start-delay=4", "--backend", "macos-subprocess"})
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}
	if opts.Rate != 3 {
		t.Fatalf("unexpected rate: %f", opts.Rate)
	}
	if opts.StartDelay != 4 {
		t.Fatalf("unexpected delay: %f", opts.StartDelay)
	}
	if opts.Backend != "macos-subprocess" {
		t.Fatalf("unexpected backend: %s", opts.Backend)
	}
}

func TestParseOptionsHelp(t *testing.T) {
	_, showHelp, err := ParseOptions([]string{"--help"})
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}
	if !showHelp {
		t.Fatalf("expected showHelp to be true")
	}
}

func TestParseOptionsRateMustBePositive(t *testing.T) {
	_, _, err := ParseOptions([]string{"--rate", "0"})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestParseOptionsUnknownFlag(t *testing.T) {
	_, _, err := ParseOptions([]string{"--nope"})
	if err == nil {
		t.Fatalf("expected unknown flag error")
	}
}
