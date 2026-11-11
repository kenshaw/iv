<div align="center">
  <img src="../png/logo.png" height="120">
</div>

<div align="center">
  <a href="#install" title="Install">Install</a> |
  <a href="#formats" title="Formats">Formats</a> |
  <a href="#notes" title="Notes">Notes</a>
</div>

<br/>

A Markdown file that starts with html, as many project READMEs do. libmagic
types it as `text/html`, and the `blitz` decoder must still read it as
**Markdown**, with the html fragments drawn in place.

## Install

Build it with `go build`, then run it on a file:

```sh
iv README.md
```

## Formats

| Format   | Extension | Notes                                        |
| -------- | --------- | -------------------------------------------- |
| Markdown | `md`      | html fragments are allowed <sup>[†][f-1]</sup> |
| html     | `html`    | a full document <sup>[‡][f-2]</sup>          |

[f-1]: #notes
[f-2]: #notes

<hr/>

## Notes

<p>
  <b>Bold</b>, <i>italic</i>, <u>underlined</u> and <code>code</code> in a
  paragraph of html.<br>
  A line after a break.
</p>

<sup>†</sup> The fragments are drawn as html.<br>
<sup>‡</sup> A document that starts with `<html>` is not Markdown.
