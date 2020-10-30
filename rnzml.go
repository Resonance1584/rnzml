package rnzml

import (
	"bufio"
	"fmt"
	"html/template"
	"io"
	"strings"
	"unicode/utf8"
)

const (
	// HTML Constants
	codeBlockStartString = "<pre><code>"
	codeBlockEndString   = "</code></pre>\n"
	textBlockStartString = "<p>"
	textBlockEndString   = "\n</p>\n"
	boldTextStartString  = "<strong>"
	boldTextEndString    = "</strong>"
	codeTextStartString  = "<code>"
	codeTextEndString    = "</code>"
	newlineString        = "\n"
)

var linkTemplate = template.Must(template.New("href").Parse(`<a href="{{.URL}}">{{.Label}}</a>`))

type link struct {
	URL   string
	Label string
}

// Renderer provides functionality to parse and render rnzml to HTML
type Renderer struct {
	codeBlockStart []byte
	codeBlockEnd   []byte
	textBlockStart []byte
	textBlockEnd   []byte
	boldTextStart  []byte
	boldTextEnd    []byte
	codeTextStart  []byte
	codeTextEnd    []byte
	newline        []byte
}

// NewRenderer returns an initialized Renderer
func NewRenderer() *Renderer {
	return &Renderer{
		codeBlockStart: []byte("<pre><code>"),
		codeBlockEnd:   []byte("</code></pre>\n"),
		textBlockStart: []byte("<p>"),
		textBlockEnd:   []byte("\n</p>\n"),
		boldTextStart:  []byte("<strong>"),
		boldTextEnd:    []byte("</strong>"),
		codeTextStart:  []byte("<code>"),
		codeTextEnd:    []byte("</code>"),
		newline:        []byte("\n"),
	}
}

// Render iterates over in line by line and either renders a text block or a
// code block
func (re *Renderer) Render(in io.Reader, out io.Writer) error {
	lineCount := 0

	codeBlockStartLine := -1
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		lineCount++
		line := scanner.Text()
		if line == "```" {
			if codeBlockStartLine == -1 {
				codeBlockStartLine = lineCount
				if _, err := out.Write(re.codeBlockStart); err != nil {
					return err
				}
			} else {
				codeBlockStartLine = -1
				if _, err := out.Write(re.codeBlockEnd); err != nil {
					return err
				}
			}
		} else {
			if codeBlockStartLine == -1 && line != "" {
				// Write a text block line
				if _, err := out.Write(re.textBlockStart); err != nil {
					return err
				}

				if err := re.renderLine(line, out); err != nil {
					return fmt.Errorf("line %d: %w", lineCount, err)
				}

				if _, err := out.Write(re.textBlockEnd); err != nil {
					return err
				}
			} else {
				// Write a code block line
				template.HTMLEscape(out, scanner.Bytes())
				if _, err := out.Write(re.newline); err != nil {
					return err
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if codeBlockStartLine != -1 {
		return fmt.Errorf("unclosed code block (```) on line: %d", codeBlockStartLine)
	}
	return nil
}

// renderLine renders a single line in a text block
func (re *Renderer) renderLine(line string, out io.Writer) error {
	// Reuse rune buffer for encoding to output
	runeBuffer := []byte{4}
	writeEscapedRune := func(r rune, out io.Writer) {
		byteCount := utf8.EncodeRune(runeBuffer, r)
		template.HTMLEscape(out, runeBuffer[:byteCount])
	}

	// Track position of last control characters for error reporting.
	// When a control character occurs again reset the value.
	lastEscape := -1
	lastBold := -1
	lastCode := -1
	lastLink := -1

	// Links are rendered using html/template to contextually escape content.
	// When the link is started runes are written to linkContent, when finished
	// linkContent is rendered to out and reset.
	linkContent := strings.Builder{}

	for n, r := range line {
		if lastEscape > -1 {
			// Always check for escape first
			if lastLink > -1 {
				linkContent.WriteRune(r) //nolint: errcheck
			} else {
				writeEscapedRune(r, out)
			}
			lastEscape = -1
		} else if lastLink > -1 {
			if r == '\\' { // Escapes still work on ] in links
				lastEscape = n
			} else if r == ']' { // End link is the only control character in a link
				lastLink = -1

				// Links are of the format [url label] where label can contain spaces
				parts := strings.SplitN(linkContent.String(), " ", 2)
				if len(parts) != 2 {
					return fmt.Errorf("Links must have a URL and a Label separated by a space. Instead found: %s", linkContent.String())
				}
				err := linkTemplate.Execute(out, link{
					URL:   parts[0],
					Label: parts[1],
				})
				if err != nil {
					return err
				}
				// Reset linkContent for next link
				linkContent = strings.Builder{}
			} else {
				// Write current rune to current link
				linkContent.WriteRune(r) //nolint: errcheck
			}
		} else if lastCode > -1 {
			if r == '\\' { // Escapes still work on `
				lastEscape = n
			} else if r == '`' { // End code is the only control character in code
				if _, err := out.Write(re.codeTextEnd); err != nil {
					return err
				}
				lastCode = -1
			} else {
				writeEscapedRune(r, out)
			}
		} else {
			switch r {
			case '\\':
				lastEscape = n
			case '*':
				if lastBold < 0 {
					if _, err := out.Write(re.boldTextStart); err != nil {
						return err
					}
					lastBold = n
				} else {
					if _, err := out.Write(re.boldTextEnd); err != nil {
						return err
					}
					lastBold = -1
				}
			case '`':
				if _, err := out.Write(re.codeTextStart); err != nil {
					return err
				}
				lastCode = n
			case '[':
				lastLink = n

			default:
				writeEscapedRune(r, out)
			}
		}
	}

	// Check for any unclosed control characters and if so return an error
	if lastBold > -1 {
		return fmt.Errorf("unclosed bold text (*) at position: %d", lastBold)
	}
	if lastCode > -1 {
		return fmt.Errorf("unclosed code text (`) at position: %d", lastCode)
	}
	if lastLink > -1 {
		return fmt.Errorf("unclosed link ([) at position: %d", lastLink)
	}
	return nil
}
