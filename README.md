# rnzml
res.nz markup language

## Usage

Parses rnzml content and outputs a subset of HTML

## Syntax

### In a text block

| Control Character | Effect |
|-------------------|--------|
| `\` | Escape the following character |
| `*` | Start or end bold text |
| `` ` `` | Start or end an inline code block |
| ```` ``` ```` | If preceded and followed by a newline start or end a code block |
| `[` | Start a Link |
| `]` | End a Link |

### Code Blocks

Code blocks do not apply any formatting to text and do not support links. It is impossible to write a line containing only ```` ``` ```` inside a code block (it will end the code block).

### Links

Links must consist of a URL and a Label separated by a single whitespace character. E.g. `[https:///res.nz/path?param=1%202 The res.nz website]` will be parsed as
```
Link {
    URL: "https:///res.nz/path?param=1%20"
    Label: "The res.nz website
}
```
Control characters other than `\` and `]` have no effect inside a link.

### Escaping HTML

Characters are passed through golang's template.HTMLEscape **except** for Links which are rendered using an html/template. Package is expected to be used on trusted input. No safety guarantees are given.

## Example

Input:
````
Here is some text

Here is *some* text with `formatting`

Here is a [url label eh] link

```
Here is some code
that is preformatted
```
````
Output:
```
<p>Here is some text
</p>
<p>Here is <strong>some</strong> text with <code>formatting</code>
</p>
<p>Here is a <a href="url>label eh</a> link
</p>
<pre><code>Here is some code
that is preformatted
</code></pre>
```
