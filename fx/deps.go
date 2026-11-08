package fx

import (
	"context"
	"io"
	"os"
	"os/exec"

	"wizki/core"
)

// MissingDeps controlla le dipendenze: un comando manca se non è nel PATH;
// una lingua di Tesseract manca se tesseract non c'è o non la elenca.
func MissingDeps(ctx context.Context, deps []core.Dep) map[string]bool {
	missing := map[string]bool{}
	var langs []string
	if _, err := exec.LookPath("tesseract"); err == nil {
		if out, err := exec.CommandContext(ctx, "tesseract", "--list-langs").CombinedOutput(); err == nil {
			langs = core.ParseLangs(string(out))
		}
	}
	for _, d := range deps {
		if d.Lang != "" {
			missing[d.Name] = len(core.MissingLangs(langs, d.Lang)) > 0
			continue
		}
		_, err := exec.LookPath(d.Name)
		missing[d.Name] = err != nil
	}
	return missing
}

// DetectManager restituisce il primo gestore di pacchetti di core.Managers
// presente nel PATH ("apt" se c'è apt-get), o "".
func DetectManager() string {
	for _, m := range core.Managers {
		bin := m
		if m == "apt" {
			bin = "apt-get"
		}
		if _, err := exec.LookPath(bin); err == nil {
			return m
		}
	}
	return ""
}

// RunInteractive lancia un comando collegato al terminale (serve a sudo per
// chiedere la password).
func RunInteractive(ctx context.Context, cmd []string, in io.Reader, out, errw io.Writer) error {
	c := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	c.Stdin, c.Stdout, c.Stderr = in, out, errw
	return c.Run()
}

// IsTerminal: lo standard input è un terminale (c'è qualcuno a cui chiedere).
// /dev/null è anch'esso un dispositivo a caratteri: va escluso a parte.
func IsTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	if null, err := os.Stat(os.DevNull); err == nil && os.SameFile(fi, null) {
		return false
	}
	return true
}
