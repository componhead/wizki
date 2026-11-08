package fx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"wizki/core"
)

// ErrOCRUnavailable: mancano pdftoppm, tesseract o il pacchetto lingua.
var ErrOCRUnavailable = errors.New("OCR non disponibile")

// CheckOCR verifica comandi e lingue PRIMA di toccare qualsiasi file.
func CheckOCR(ctx context.Context, lang string) error {
	for _, bin := range []string{"pdftoppm", "tesseract"} {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("%w: manca %s (servono poppler e tesseract)", ErrOCRUnavailable, bin)
		}
	}
	out, err := exec.CommandContext(ctx, "tesseract", "--list-langs").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: tesseract --list-langs: %v", ErrOCRUnavailable, err)
	}
	if miss := core.MissingLangs(core.ParseLangs(string(out)), lang); len(miss) > 0 {
		return fmt.Errorf("%w: manca il pacchetto lingua %s di Tesseract", ErrOCRUnavailable, strings.Join(miss, ", "))
	}
	return nil
}

// stderrOf aggiunge lo stderr del comando all'errore, se disponibile.
func stderrOf(name string, err error) error {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(string(ee.Stderr)))
	}
	return fmt.Errorf("%s: %w", name, err)
}

// execOCR: pdftoppm a 300 dpi in una cartella temporanea, poi tesseract su
// ogni pagina, nell'ordine corretto.
func execOCR(ctx context.Context, s core.OCR) error {
	tmp, err := os.MkdirTemp("", "wiki-ocr-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	prefix := filepath.Join(tmp, "p")
	if out, err := exec.CommandContext(ctx, "pdftoppm", "-r", "300", "-png", s.PDF, prefix).CombinedOutput(); err != nil {
		return fmt.Errorf("pdftoppm: %w: %s", err, strings.TrimSpace(string(out)))
	}
	imgs, err := filepath.Glob(prefix + "-*.png")
	if err != nil {
		return err
	}
	if len(imgs) == 0 {
		return errors.New("pdftoppm non ha prodotto nessuna pagina")
	}
	var texts []string
	for _, img := range core.SortPages(imgs) {
		out, err := exec.CommandContext(ctx, "tesseract", img, "stdout", "-l", s.Lang).Output()
		if err != nil {
			return stderrOf("tesseract", err)
		}
		texts = append(texts, string(out))
	}
	text := core.JoinPages(texts)
	if text == "" {
		return ErrNoText
	}
	return os.WriteFile(s.Out, []byte(text), 0o644)
}

// CheckTesseract verifica tesseract e le lingue richieste: basta per l'OCR
// delle immagini, che non passa da pdftoppm.
func CheckTesseract(ctx context.Context, lang string) error {
	if _, err := exec.LookPath("tesseract"); err != nil {
		return fmt.Errorf("%w: manca tesseract", ErrOCRUnavailable)
	}
	out, err := exec.CommandContext(ctx, "tesseract", "--list-langs").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: tesseract --list-langs: %v", ErrOCRUnavailable, err)
	}
	if miss := core.MissingLangs(core.ParseLangs(string(out)), lang); len(miss) > 0 {
		return fmt.Errorf("%w: manca il pacchetto lingua %s di Tesseract", ErrOCRUnavailable, strings.Join(miss, ", "))
	}
	return nil
}

// execImageText scrive il riferimento all'immagine e, se richiesto, il testo
// letto da Tesseract. Un'immagine senza testo non è un errore: la si guarda.
func execImageText(ctx context.Context, s core.ImageText) error {
	text := s.Header
	if s.Lang != "" {
		out, err := exec.CommandContext(ctx, "tesseract", s.Image, "stdout", "-l", s.Lang).Output()
		if err != nil {
			return stderrOf("tesseract", err)
		}
		if t := strings.TrimSpace(string(out)); t != "" {
			text += "\n## Testo (OCR)\n\n" + t + "\n"
		}
	}
	return Exec(ctx, core.WriteFile{Path: s.Out, Data: []byte(text)})
}
